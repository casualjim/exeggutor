// Package main provides ...
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	http.HandleFunc("/", helloWold)
	addr := fmt.Sprintf(":%v", readPort(3000))
	log.Println("Starting webserver at", addr)
	http.ListenAndServe(addr, nil)
}

func readPort(def int) int {
	port := os.Getenv("PORT")
	if port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			log.Fatalf("The value of the port environment variable is %v which is not convertible to int", port)
		}
		return p
	}
	return def
}

func helloWold(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World"))
}
