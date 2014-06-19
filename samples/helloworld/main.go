// Package main provides ...
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", helloWold)
	addr := fmt.Sprintf(":%v", 3000)
	log.Println("Starting webserver at", addr)
	http.ListenAndServe(addr, nil)
}

func helloWold(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World"))
}
