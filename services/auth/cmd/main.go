package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Auth Service")
	})
	log.Println("Auth Service starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
