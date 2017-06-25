package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("Debug Application Staring")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	log.Fatalln(http.ListenAndServe(":8080", nil))
}
