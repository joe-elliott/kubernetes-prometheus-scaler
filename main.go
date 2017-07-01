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

	"github.com/Knetic/govaluate"
	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
)

const DeploymentLabelSelector = "scale==prometheus"

const DeploymentAnnotationPrometheusQuery = "prometheusScaler/prometheus-query"
const DeploymentAnnotationMinScale = "prometheusScaler/min-scale"
const DeploymentAnnotationMaxScale = "prometheusScaler/max-scale"
const DeploymentAnnotationScaleUpWhen = "prometheusScaler/scale-up-when"
const DeploymentAnnotationScaleDownWhen = "prometheusScaler/scale-down-when"

func main() {

	clientURL := "http://prometheus:9090"

	promQuery, err := makeQueryFunc(clientURL)
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

			query := deployment.Annotations[DeploymentAnnotationPrometheusQuery]
			minScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMinScale], 10, 32)
			maxScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMaxScale], 10, 32)
			scaleUpWhen := deployment.Annotations[DeploymentAnnotationScaleUpWhen]
			scaleDownWhen := deployment.Annotations[DeploymentAnnotationScaleDownWhen]

			replicaCount := deployment.Spec.Replicas

			fmt.Printf("current replica count: %v \n", replicaCount)
			fmt.Printf("query: %v \n", query)

			val, err := promQuery(query)

			if err != nil {
				fmt.Printf("err: %v\n", err)
			}

			fmt.Printf("val: %f \n", val)
			fmt.Printf("scaleUpWhen: %v \n", scaleUpWhen)
			fmt.Printf("scaleDownWhen: %v \n", scaleDownWhen)

			strVal := strconv.FormatFloat(val, 'f', -1, 64)
			exprScaleUpWhen, err := govaluate.NewEvaluableExpression(strVal + scaleUpWhen)
			exprScaleDownWhen, err := govaluate.NewEvaluableExpression(strVal + scaleDownWhen)

			scaleUp, err := exprScaleUpWhen.Evaluate(nil)
			scaleDown, err := exprScaleDownWhen.Evaluate(nil)

			fmt.Printf("scaleUp: %v \n", scaleUp)
			fmt.Printf("scaleDown: %v \n", scaleDown)

			if scaleUp == true && *replicaCount < int32(maxScale) {
				*replicaCount++
			}
			if scaleDown == true && *replicaCount > int32(minScale) {
				*replicaCount--
			}

			//todo figure out how to do this
			fmt.Printf("Setting replica count to %d\n", *replicaCount)
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
