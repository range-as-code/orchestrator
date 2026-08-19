package main

import (
	"fmt"
	"log"
	"os"

	"github.com/range-as-code/orchestrator/guac"
)

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func main() {
	client := guac.NewGuacClient(mustEnv("GUAC_BASE_URL"))
	if err := client.Authenticate(mustEnv("GUAC_USERNAME"), mustEnv("GUAC_PASSWORD")); err != nil {
		log.Fatalf("authentication failed: %v", err)
	}
	fmt.Println("token:", client.Token)
}
