package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"

	"os"

	"github.com/Knetic/govaluate"
	"github.com/op/go-logging"
	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
)

const AppName = "prometheus-autoscaler"

const DeploymentLabelSelector = "scale==prometheus"

const DeploymentAnnotationPrometheusQuery = "prometheusScaler/prometheus-query"
const DeploymentAnnotationMinScale = "prometheusScaler/min-scale"
const DeploymentAnnotationMaxScale = "prometheusScaler/max-scale"
const DeploymentAnnotationScaleUpWhen = "prometheusScaler/scale-up-when"
const DeploymentAnnotationScaleDownWhen = "prometheusScaler/scale-down-when"

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

			// parse scaling parameters from deployment spec
			query := deployment.Annotations[DeploymentAnnotationPrometheusQuery]
			scaleUpWhen := deployment.Annotations[DeploymentAnnotationScaleUpWhen]
			scaleDownWhen := deployment.Annotations[DeploymentAnnotationScaleDownWhen]
			minScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMinScale], 10, 32)

			if err != nil {
				log.Errorf("  minScale: %v", err)
				continue
			}

			maxScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMaxScale], 10, 32)

			if err != nil {
				log.Errorf("  maxScale: %v", err)
				continue
			}

			// get current state
			replicaCount := int64(*deployment.Spec.Replicas)

			log.Infof("  current replica count: %v", replicaCount)
			log.Infof("  query: %v", query)

			// get and evaluate promQuery
			result, err := promQuery(query)

			if err != nil {
				log.Errorf("promQuery: %v", err)
				continue
			}

			log.Infof("  result: %f", result)
			log.Infof("  scaleUpWhen: %v", scaleUpWhen)
			log.Infof("  scaleDownWhen: %v", scaleDownWhen)

			parameters := make(map[string]interface{}, 1)
			parameters["result"] = result
			exprScaleUpWhen, err := govaluate.NewEvaluableExpression(scaleUpWhen)

			if err != nil {
				log.Errorf("  exprScaleUpWhen: %v", err)
				continue
			}

			exprScaleDownWhen, err := govaluate.NewEvaluableExpression(scaleDownWhen)

			if err != nil {
				log.Errorf("  exprScaleDownWhen: %v", err)
				continue
			}

			scaleUp, err := exprScaleUpWhen.Evaluate(parameters)

			if err != nil {
				log.Errorf("  exprScaleUpWhen: %v", err)
				continue
			}

			scaleDown, err := exprScaleDownWhen.Evaluate(parameters)

			if err != nil {
				log.Errorf("  exprScaleDownWhen: %v", err)
				continue
			}

			// scale up or down
			log.Infof("  scaleUp: %v", scaleUp)
			log.Infof("  scaleDown: %v", scaleDown)

			if scaleUp == true && replicaCount < maxScale {
				replicaCount++
			}
			if scaleDown == true && replicaCount > minScale {
				replicaCount--
			}

			// set the replica set
			log.Infof("  Setting replica count to %d\n", replicaCount)
			jsonPatch := "[{\"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": " + strconv.FormatInt(replicaCount, 10) + " }]"
			log.Infof("  Patch string: %v\n", jsonPatch)
			_, err = clientset.Extensions().Deployments(deployment.Namespace).Patch(deployment.Name, api.JSONPatchType, []byte(jsonPatch))

			if err != nil {
				log.Errorf("  Error scaling: %v", err)
				continue
			}
		}

		/*
			// Examples for error handling:
			// - Use helper functions like e.g. errors.IsNotFound()
			// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
			_, err = clientset.CoreV1().Pods("default").Get("example-xxxxx")
			if errors.IsNotFound(err) {
				fmt.Printf("Pod not found\n")
			} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
				fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
			} else if err != nil {
				panic(err.Error())
			} else {
				fmt.Printf("Found pod\n")
			}
		*/
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
