package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const defaultConfFile = "settings.yaml"
const defaultPort = "8080"

type msg struct {
	Message string `json:"msg"` //= {"msg":"Hi"}
}

func main() {

	var confFile = defaultConfFile
	var port = defaultPort

	/*
		Usage: cw-exporter <conf_file> <port>
		Defaults:
			conf_file: settings.yaml
			port: 8080
	*/

	if len(os.Args) > 1 {
		confFile = os.Args[1]
	}

	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	log.Printf("Loading config file %s", confFile)
	log.Printf("Opening port %s", port)

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(&msg{Message: "Hello Golang UK Conf"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
