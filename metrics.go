package main

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"log"

	"regexp"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func metricsEndpoint(cfg GlobalConfig) func(http.ResponseWriter, *http.Request) {

	sess := session.Must(session.NewSession())
	svc := cloudwatch.New(sess)

	return func(w http.ResponseWriter, r *http.Request) {

		hasError := false

		for _, metricCfg := range cfg.Metrics {

			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:          &metricCfg.Namespace,
				MetricName:         &metricCfg.Name,
				Dimensions:         nil,
				StartTime:          nil,
				EndTime:            nil,
				Period:             nil,
				Statistics:         nil,
				ExtendedStatistics: nil,
			}
			output, err := svc.GetMetricStatistics(input)

			if err != nil {
				hasError = true
				log.Printf("Error Requesting input: %+v err: %+v", input, err)
			} else {
				labels := map[string]string{
					"job": safeName(metricCfg.Namespace),
				}

				writeSingleMetric(getMetricName(metricCfg), labels, strconv.FormatFloat(*output.Datapoints[0].Sum, 'f', -1, 64), w)
			}
		}

		//error
		if hasError {
			writeSingleMetric("cloudwatch_exporter_scrape_error", nil, "1", w)
		} else {
			writeSingleMetric("cloudwatch_exporter_scrape_error", nil, "0", w)
		}
	}
}

func getMetricName(cfg MetricConfig) string {
	return safeName(cfg.Namespace + "_" + toSnakeCase(cfg.Name))
}

/*
	todo: make these more efficient
*/
func safeName(s string) string {
	reg1, _ := regexp.Compile("[^a-zA-Z0-9:_]")
	reg2, _ := regexp.Compile("__+")

	s = strings.ToLower(s)

	s = reg1.ReplaceAllString(s, "_")
	s = reg2.ReplaceAllString(s, "_")

	return s
}

func toSnakeCase(s string) string {
	reg, _ := regexp.Compile("([a-z0-9])([A-Z])")

	s = reg.ReplaceAllString(s, "$1_$2")

	return s
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

		buffer.WriteString("}")
	}

	buffer.WriteString(" ")
	buffer.WriteString(value)
	buffer.WriteString("\n")

	writer.Write(buffer.Bytes())
}
