package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	f, err := os.Open(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	type label struct {
		Name string `json:"name"`
	}
	var event struct {
		PullRequest struct {
			Labels []label
		} `json:"pull_request"`
	}
	if err := json.NewDecoder(f).Decode(&event); err != nil {
		log.Fatal(err)
	}

	// Check for any "conflicts" and "backport-*" labels.
	var haveBackport bool
	for _, label := range event.PullRequest.Labels {
		if label.Name == "conflict" {
			fmt.Printf("::error::%s\n", "Cannot merge with 'conflict' label")
			os.Exit(1)
		}
		if !haveBackport && strings.HasPrefix(label.Name, "backport-") {
			haveBackport = true
		}
	}
	if !haveBackport {
		fmt.Printf("::error::%s\n", "Missing 'backport-*' label")
		os.Exit(1)
	}
}
