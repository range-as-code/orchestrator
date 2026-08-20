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
		Name:             "test box Hello",
		Protocol:         "ssh",
		Hostname:         "127.112.0.0",
		Port:             "22",
		Username:         "changeme",
		Password:         "changeme",
	})
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	fmt.Println("connection id returned: ", got)

	// err = client.DeleteConnection(got)
	// if err != nil {
	// 	log.Fatalf("delete connection: %v", err)
	// }

}
