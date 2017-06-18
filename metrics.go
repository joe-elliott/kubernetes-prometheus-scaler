package main

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"log"

	"regexp"

	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func metricsEndpoint(cfg GlobalConfig) func(http.ResponseWriter, *http.Request) {

	sess := session.Must(session.NewSession())

	sess.Config.Region = &cfg.Region

	svc := cloudwatch.New(sess)

	return func(w http.ResponseWriter, r *http.Request) {

		hasError := false
		start := time.Now()

		for _, metricCfg := range cfg.Metrics {

			periodSeconds := getConfigValue(cfg.PeriodSeconds, metricCfg.PeriodSeconds)
			delaySeconds := getConfigValue(cfg.DelaySeconds, metricCfg.DelaySeconds)
			rangeSeconds := getConfigValue(cfg.RangeSeconds, metricCfg.RangeSeconds)

			now := time.Now()
			startTime := now.Add(time.Duration(delaySeconds) * time.Second)
			endTime := now.Add(time.Duration(delaySeconds+rangeSeconds) * time.Second)
			period := int64(periodSeconds)

			dimensions := []*cloudwatch.Dimension{}

			for _, dim := range metricCfg.Dimensions {
				dimensions = append(dimensions, &cloudwatch.Dimension{
					Name:  &dim,
					Value: nil,
				})
			}

			statistics := []*string{}
			for _, stat := range metricCfg.Statistics {
				statistics = append(statistics, &stat)
			}

			input := &cloudwatch.GetMetricStatisticsInput{
				Namespace:          &metricCfg.Namespace,
				MetricName:         &metricCfg.Name,
				Dimensions:         nil,
				StartTime:          &startTime,
				EndTime:            &endTime,
				Period:             &period,
				Statistics:         statistics,
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

				for _, datapoint := range output.Datapoints {
					writeSingleMetric(getMetricName(metricCfg), labels, getMetricVal(*datapoint.Minimum), w)
				}
			}
		}

		//scrape time
		writeSingleMetric("cloudwatch_exporter_scrape_duration_seconds", nil, getMetricVal(time.Since(start).Seconds()), w)

		//error
		if hasError {
			writeSingleMetric("cloudwatch_exporter_scrape_error", nil, "1", w)
		} else {
			writeSingleMetric("cloudwatch_exporter_scrape_error", nil, "0", w)
		}
	}
}

func getMetricVal(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
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

func getConfigValue(global int, local int) int {
	if local == -1 {
		return global
	}

	return local
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
