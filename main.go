package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"

	"github.com/op/go-logging"
	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"

	"kubernetes-prometheus-scaler/scaler"
)

const DeploymentLabelSelector = "scale==prometheus"

var log = logging.MustGetLogger("prometheus-autoscaler")

var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{message}`,
)

var prometheusURL = flag.String("prometheus-url", "http://prometheus:9090", "URL to query.")
var assessmentInterval = flag.Duration("assessment-interval", 60*time.Second, "Time to sleep between checking deployments.")

func main() {

	flag.Parse()

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatted := logging.NewBackendFormatter(backend, format)

	logging.SetBackend(backendFormatted)

	log.Infof("prometheus-url=%v", *prometheusURL)
	log.Infof("assessment-interval=%v", *assessmentInterval)

	promQuery, err := makeQueryFunc(*prometheusURL)
	if err != nil {
		log.Criticalf("makeQueryFunc failed: %v", err)
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Criticalf("rest.InClusterConfig failed: %v", err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Criticalf("kubernetes.NewForConfig failed: %v", err)
	}

	for {

		deployments, err := clientset.Extensions().Deployments("").List(v1.ListOptions{
			LabelSelector: DeploymentLabelSelector,
		})
		if err != nil {
			log.Errorf("list deployments: %v", err)
			continue
		}

		log.Infof("Considering %d deployments for scaling.", len(deployments.Items))

		for _, deployment := range deployments.Items {

			log.Infof("Considering: %v", deployment.Name)

			scalable, err := scaler.NewScalable(deployment)

			if err != nil {
				log.Errorf("NewScalable: %v", err)
				continue
			}

			log.Infof("Scalable: %+v", scalable)

			scalingFunc, err := scaler.MakeScalingFunc(scalable)

			if err != nil {
				log.Errorf("scalingFunc: %v", err)
				continue
			}

			// get and evaluate promQuery
			result, err := promQuery(scalable.GetQuery())

			if err != nil {
				log.Errorf("promQuery: %v", err)
				continue
			}

			log.Infof("result: %f", result)

			newScale, err := scalingFunc(result)

			if err != nil {
				log.Errorf("scalingFunc: %v", err)
				continue
			}

			// set the replica set
			if scalable.GetCurScale() != newScale {
				log.Infof("Setting replica count to %d\n", newScale)
				jsonPatch := "[{\"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": " + strconv.FormatInt(newScale, 10) + " }]"
				log.Infof("Patch string: %v\n", jsonPatch)
				_, err = clientset.Extensions().Deployments(deployment.Namespace).Patch(deployment.Name, api.JSONPatchType, []byte(jsonPatch))

				if err != nil {
					log.Errorf("  Error scaling: %v", err)
					continue
				}
			} else {
				log.Infof("curScale == newScale.  Not scaling %v", deployment.Name)
			}
		}

		time.Sleep(*assessmentInterval)
	}
}

func makeQueryFunc(url string) (func(query string) (float64, error), error) {

	client, err := prometheus.New(prometheus.Config{
		Address: url,
	})

	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, errors.New("client is nil")
	}

	api := prometheus.NewQueryAPI(client)

	if api == nil {
		return nil, errors.New("api is nil")
	}

	return func(query string) (float64, error) {

		val, err := api.Query(context.Background(), query, time.Now())

		if err != nil {
			return 0, err
		}

		if val == nil {
			return 0, errors.New("val is nil")
		}

		original, ok := val.(*model.Scalar)

		if !ok {
			return 0, fmt.Errorf("not a scalar %v", val)
		}

		res, err := strconv.ParseFloat(original.Value.String(), 64)

		if err != nil {
			return 0, err
		}

		return res, nil
	}, nil
}
