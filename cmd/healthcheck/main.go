package main

import (
	"net/http"
	"os"
)

func main() {
	resp, err := http.Get("http://localhost:8080/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
