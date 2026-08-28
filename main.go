package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/range-as-code/orchestrator/guac"
	"github.com/range-as-code/orchestrator/tofu"
)

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func testGuacIntegration() {

	client := guac.NewGuacClient(mustEnv("GUAC_BASE_URL"))
	if err := client.Authenticate(mustEnv("GUAC_USERNAME"), mustEnv("GUAC_PASSWORD")); err != nil {
		log.Fatalf("authentication failed: %v", err)
	}
	fmt.Println("token:", client.Token)

	fmt.Println("using token to create connection...")
	fmt.Println("creating connection group")

	groupID, err := client.CreateConnectionGroup(guac.ConnectionGroupSpec{
		ParentIdentifier: "ROOT",
		Name:             "Group A",
		Type:             "ORGANIZATIONAL",
	})
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	fmt.Println("Group created:", groupID)

	got, err := client.CreateConnection(guac.ConnectionSpec{
		ParentIdentifier: groupID,
		Name:             "test box HelloWorld",
		Protocol:         "ssh",
		Hostname:         "localhost",
		Port:             "22",
		Username:         "changeme",
		Password:         "changeme",
	})
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	fmt.Println("connection id returned: ", got)

	deleted, err := client.DeleteGroup(groupID)
	if err != nil {
		log.Fatalf("delete group: %v", err)
	}

	fmt.Println("group deleted:", deleted)

	deleted, err = client.DeleteConnection(got)
	if err != nil {
		log.Fatalf("delete connection: %v", err)
	}

	fmt.Println("connection deleted:", deleted)

}

func testTofuIntegration() {
	scenarios := map[string]tofu.Scenario{
		"box5-b": {
			Repo: "https://github.com/range-as-code/helloNix.git",
		},
	}

	runner, err := tofu.NewTofuRunner()
	if err != nil {
		log.Fatalf("create runner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := runner.Init(ctx, "box5b", scenarios, "./tofu/workspace1/main.tf"); err != nil {
		log.Fatalf("tofu init: %v", err)
	}
	log.Printf("init returned: %v", err)

	if err := runner.Apply(ctx, "box5b"); err != nil {
		log.Fatalf("tofu apply: %v", err)
	}

	creds, err := runner.Output(ctx, "box5b")
	if err != nil {
		log.Fatalf("tofu output: %v", err)
	}
	tofu.PrintCredentials(creds)
}

func testTofuOutputIntegration() {
	runner, err := tofu.NewTofuRunner()
	if err != nil {
		log.Fatalf("create runner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	creds, err := runner.Output(ctx, "box5b")
	if err != nil {
		log.Fatalf("tofu output: %v", err)
	}
	tofu.PrintCredentials(creds)

}

func testTofuDestroyIntegration() {
	runner, err := tofu.NewTofuRunner()
	if err != nil {
		log.Fatalf("create runner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err = runner.Destroy(ctx, "box5b")
	if err != nil {
		log.Fatalf("tofu destroy: %v", err)
	}

}
func main() {
	//testGuacIntegration()
	testTofuIntegration()
	testTofuOutputIntegration()
	testTofuDestroyIntegration()
}
