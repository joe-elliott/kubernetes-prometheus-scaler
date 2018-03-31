package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"

	"github.com/op/go-logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"kubernetes-prometheus-scaler/util"
)

const (
	deploymentLabelSelector = "scale==prometheus"
	prometheusNamespace     = "prometheusscaler"
)

var (
	_log       = logging.MustGetLogger("prometheus-autoscaler")
	_logFormat = logging.MustStringFormatter(
		`%{time:15:04:05.000} %{level:.4s} %{message}`,
	)

	_prometheusURL      = flag.String("prometheus-url", "http://prometheus:9090", "URL to query.")
	_assessmentInterval = flag.Duration("assessment-interval", 60*time.Second, "Time to sleep between checking deployments.")

	_errorTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: "error",
			Name:      "total",
			Help:      "Kubernetes Prometheus Scaler Errors",
		},
	)
)

func init() {
	prometheus.MustRegister(_errorTotal)

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatted := logging.NewBackendFormatter(backend, _logFormat)

	logging.SetBackend(backendFormatted)
}

func main() {

	flag.Parse()

	_log.Infof("prometheus-url=%v", *_prometheusURL)
	_log.Infof("assessment-interval=%v", *_assessmentInterval)

	prometheusQuery, err := util.MakePrometheusQueryFunc(*_prometheusURL)
	if err != nil {
		_log.Criticalf("makeQueryFunc failed: %v", err)
		os.Exit(1)
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		_log.Criticalf("rest.InClusterConfig failed: %v", err)
		os.Exit(1)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		_log.Criticalf("kubernetes.NewForConfig failed: %v", err)
		os.Exit(1)
	}

	go func() {
		for {

			deployments, err := clientset.Extensions().Deployments("").List(v1.ListOptions{
				LabelSelector: deploymentLabelSelector,
			})
			if err != nil {
				_errorTotal.Inc()
				_log.Errorf("list deployments: %v", err)
				continue
			}

			_log.Infof("Considering %d deployments for scaling.", len(deployments.Items))

			for _, deployment := range deployments.Items {

				_log.Infof("Considering: %v", deployment.Name)

				scalable, err := util.NewScalable(deployment)

				if err != nil {
					_errorTotal.Inc()
					_log.Errorf("NewScalable: %v", err)
					continue
				}

				_log.Debugf("Scalable: %+v", scalable)

				// get and evaluate promQuery
				result, err := prometheusQuery(scalable.GetQuery())

				if err != nil {
					_errorTotal.Inc()
					_log.Errorf("prometheusQuery: %v", err)
					continue
				}

				_log.Infof("result: %f", result)

				newScale, err := util.CalculateNewScale(scalable, result)

				if err != nil {
					_errorTotal.Inc()
					_log.Errorf("scalingFunc: %v", err)
					continue
				}

				// set the replica set
				if scalable.GetCurScale() != newScale {
					_log.Infof("Setting replica count to %d\n", newScale)
					jsonPatch := "[{\"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": " + strconv.FormatInt(newScale, 10) + " }]"
					_log.Debugf("Patch string: %v\n", jsonPatch)
					_, err = clientset.Extensions().Deployments(deployment.Namespace).Patch(deployment.Name, api.JSONPatchType, []byte(jsonPatch))

					if err != nil {
						_errorTotal.Inc()
						_log.Errorf("  Error scaling: %v", err)
						continue
					}
				} else {
					_log.Infof("curScale == newScale.  Not scaling %v", deployment.Name)
				}
			}

			time.Sleep(*_assessmentInterval)
		}
	}()

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
