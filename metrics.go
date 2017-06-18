package main

import (
	"bytes"
	"net/http"
	"strings"
)

func metricsEndpoint(cfg GlobalConfig) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		labels := map[string]string{
			"label1": "a",
			"label2": "a",
		}

		writeSingleMetric("name", labels, "value", w)
	}
}

func writeSingleMetric(name string, labels map[string]string, value string, writer http.ResponseWriter) {

	var buffer bytes.Buffer

	buffer.WriteString(name)

	if len(labels) > 0 {
		buffer.WriteString("{")

		kvp := []string{}
		for k, v := range labels {
			kvp = append(kvp, k+"=\""+v+"\"")
		}

		buffer.WriteString(strings.Join(kvp, ","))

		buffer.WriteString("} ")
	}

	buffer.WriteString(value)
	buffer.WriteString("\n")

	writer.Write(buffer.Bytes())
}
