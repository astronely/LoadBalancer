package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

const (
	portName = "PORT"
)

var port string

func main() {
	flag.StringVar(&port, "port", "9000", "backend port")
	flag.Parse()

	if len(port) == 0 {
		log.Fatalf("Environment variable %s is not set", portName)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "Hello from backend at port %s\n", port)
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
		log.Print("Request from:", r.RemoteAddr)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Print("Health check from:", r.RemoteAddr)
	})

	log.Printf("Listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
