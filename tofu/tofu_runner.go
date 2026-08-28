package tofu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	InstancesPath      = "./instances/"
	ErrIDTaken         = errors.New("id taken")
	ErrCopyFailed      = errors.New("file copy failed")
	ErrOperationFailed = errors.New("operation failed")
)

type TofuRunner struct {
	tofuBin        string
	pluginCacheDir string
	instancesPath  string
}

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func envOr(key, or string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return or
	}

	return v
}

func NewTofuRunner() (*TofuRunner, error) {
	cache, err := filepath.Abs("./.plugin-cache")
	if err != nil {
		return nil, fmt.Errorf("resolve plugin cache path: %w", err)
	}
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	abs, err := filepath.Abs("./instances")
	if err != nil {
		return nil, fmt.Errorf("resolve instances path: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create instances dir: %w", err)
	}

	return &TofuRunner{
		tofuBin:        envOr("TOFU_BIN", "tofu"),
		pluginCacheDir: cache,
		instancesPath:  abs,
	}, nil
}

func copyFile(src, dst string, buf []byte) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.CopyBuffer(out, in, buf); err != nil {
		return err
	}
	return out.Close()
}

func (r *TofuRunner) Apply(
	ctx context.Context,
	id string,
) error {
	instanceDir := filepath.Join(r.instancesPath, id)

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, r.tofuBin, "apply", "-auto-approve")
	cmd.Dir = instanceDir
	cmd.Env = append(
		os.Environ(),
		"TF_PLUGIN_CACHE_DIR="+r.pluginCacheDir,
	)
	cmd.Stdout = io.MultiWriter(&buf, os.Stderr)
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(instanceDir)
		return fmt.Errorf("%w: %s", ErrOperationFailed, buf.String())
	}

	return nil
}

type Scenario struct {
	Repo string `json:"repo"`
}

type tfVars struct {
	Scenarios map[string]Scenario `json:"scenarios"`
}

// Note: Function does not do fsync
func writeVars(instanceDir string, scenarios map[string]Scenario) error {
	data, err := json.MarshalIndent(tfVars{Scenarios: scenarios}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tfvars: %w", err)
	}

	final := filepath.Join(instanceDir, "terraform.tfvars.json")
	tmp := final + ".tmp"

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp tfvars: %w", err)
	}

	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename tfvars: %w", err)
	}
	return nil
}

func (r *TofuRunner) Init(
	ctx context.Context,
	id string,
	scenarios map[string]Scenario,
	schemaPath string,
) error {
	instanceDir := filepath.Join(r.instancesPath, id)

	if err := os.Mkdir(instanceDir, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrIDTaken
		}
		return fmt.Errorf("mkdir instance dir: %w", err)
	}

	if err := writeVars(instanceDir, scenarios); err != nil {
		return fmt.Errorf("write tfvars: %w", err)
	}

	if err := copyFile(
		schemaPath,
		filepath.Join(instanceDir, "main.tf"),
		nil,
	); err != nil {
		_ = os.RemoveAll(instanceDir)
		return fmt.Errorf("%w: %v", ErrCopyFailed, err)
	}

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, r.tofuBin, "init", "-upgrade")
	cmd.Dir = instanceDir
	cmd.Env = append(os.Environ(), "TF_PLUGIN_CACHE_DIR="+r.pluginCacheDir)
	cmd.Stdout = io.MultiWriter(&buf, os.Stderr)
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(instanceDir)
		return fmt.Errorf("%w: %s", ErrOperationFailed, buf.String())
	}

	return nil
}

type HostCreds struct {
	IP       [][]string `json:"ip"`
	Username string     `json:"username"`
	Password string     `json:"password"`
}

// envelope: one entry per output name
type outputEnvelope struct {
	Credentials struct {
		Value map[string]HostCreds `json:"value"`
	} `json:"credentials"`
}

func (r *TofuRunner) Output(ctx context.Context, id string) (map[string]HostCreds, error) {
	instanceDir := filepath.Join(r.instancesPath, id)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, r.tofuBin, "output", "-json")
	cmd.Dir = instanceDir
	cmd.Env = append(os.Environ(), "TF_PLUGIN_CACHE_DIR="+r.pluginCacheDir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOperationFailed, stderr.String())
	}

	var env outputEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		return nil, fmt.Errorf("parse tofu output: %w", err)
	}
	return env.Credentials.Value, nil
}

func (r *TofuRunner) Destroy(ctx context.Context, id string) error {
	instanceDir := filepath.Join(r.instancesPath, id)

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, r.tofuBin, "destroy", "-auto-approve")
	cmd.Dir = instanceDir
	cmd.Env = append(
		os.Environ(),
		"TF_PLUGIN_CACHE_DIR="+r.pluginCacheDir,
	)
	cmd.Stdout = io.MultiWriter(&buf, os.Stderr)
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", ErrOperationFailed, buf.String())
	}

	_ = os.RemoveAll(instanceDir)

	return nil

}

func PrintCredentials(creds map[string]HostCreds) {
	if len(creds) == 0 {
		fmt.Println("(no credentials — output empty)")
		return
	}
	for name, c := range creds {
		ip, err := FirstUsableIP(c.IP)
		if err != nil {
			fmt.Printf(err.Error())
		}
		fmt.Printf("host=%s user=%s pass=%s ip=%s\n",
			name, c.Username, c.Password, ip)
	}
}

func FirstUsableIP(ips [][]string) (string, error) {
	for _, nic := range ips {
		for _, addr := range nic {
			ip := net.ParseIP(addr)
			if ip == nil {
				continue
			}
			if ip.IsLoopback() {
				continue
			}
			if ip.IsLinkLocalUnicast() {
				continue
			}
			return addr, nil
		}
	}
	return "", fmt.Errorf("no usable IP")
}
