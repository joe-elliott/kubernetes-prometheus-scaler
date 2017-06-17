package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const defaultConfFile = "settings.yaml"
const defaultPort = "8080"

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

	conf, err := loadConf(confFile)

	if err != nil {
		log.Fatalf("Failed to load conf.  Aborting startup. #%v", err)
		return
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(conf.region))
		//http.Error(w, err.Error(), http.StatusInternalServerError)
	})

	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
