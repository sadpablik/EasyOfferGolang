package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Question Service")
	})
	log.Println("Question Service starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
