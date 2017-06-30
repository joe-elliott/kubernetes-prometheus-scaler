package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"

	"github.com/prometheus/client_golang/api/prometheus"
)

const DeploymentLabelSelector = "scale==prometheus"
const DeploymentAnnotationPrometheusQuery = "prometheusScaler/prometheus-query"
const DeploymentAnnotationMinScale = "prometheusScaler/min-scale"
const DeploymentAnnotationMaxScale = "prometheusScaler/max-scale"

func main() {

	clientURL := "http://prometheus:9090"
	query, err := makeQueryFunc(clientURL)
	if err != nil {
		panic(err.Error())
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		deployments, err := clientset.Extensions().Deployments("").List(v1.ListOptions{
			LabelSelector: DeploymentLabelSelector,
		})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("Found %d deployments\n", len(deployments.Items))

		for _, deployment := range deployments.Items {
			// element is the element from someSlice for where we are
			fmt.Printf("name: %v\n", deployment.Name)
		}

		val, err := query("time() % (60 * 60)")

		if err != nil {
			fmt.Printf("err: %v\n", err)
		} else {
			fmt.Printf("query: %f \n", val)
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
		time.Sleep(10 * time.Second)
	}
}

func makeQueryFunc(url string) (func(query string) (float64, error), error) {
	client, err := prometheus.New(prometheus.Config{})

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

		res, err := strconv.ParseFloat(val.String(), 64)

		if err != nil {
			return 0, err
		}

		return res, nil
	}, nil
}
