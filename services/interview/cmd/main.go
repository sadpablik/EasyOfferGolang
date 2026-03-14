package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Interview Service")
	})
	log.Println("Interview Service starting on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}
