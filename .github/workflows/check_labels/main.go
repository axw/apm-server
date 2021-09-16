package main

import (
	"io"
	"log"
	"os"
)

func main() {
	f, err := os.Open(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// TODO(axw) unmarshal, check labels
	io.Copy(os.Stdout, f)
}
