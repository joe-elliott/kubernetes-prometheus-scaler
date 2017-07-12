package scaler

import (
	"fmt"
	"strconv"

	"github.com/Knetic/govaluate"
	logging "github.com/op/go-logging"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const DeploymentAnnotationPrometheusQuery = "prometheusScaler/prometheus-query"
const DeploymentAnnotationMinScale = "prometheusScaler/min-scale"
const DeploymentAnnotationMaxScale = "prometheusScaler/max-scale"
const DeploymentAnnotationScaleUpWhen = "prometheusScaler/scale-up-when"
const DeploymentAnnotationScaleDownWhen = "prometheusScaler/scale-down-when"

var log = logging.MustGetLogger("prometheus-autoscaler")

func MakeScalingFunc(deployment v1beta1.Deployment) (func(result float64) (int64, error), string, error) {

	// parse scaling parameters from deployment spec
	query := deployment.Annotations[DeploymentAnnotationPrometheusQuery]
	minScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMinScale], 10, 32)

	if err != nil {
		return nil, "", fmt.Errorf("minScale: %v", err)
	}

	maxScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMaxScale], 10, 32)

	if err != nil {
		return nil, "", fmt.Errorf("maxScale: %v", err)
	}

	// get current state
	curScale := int64(*deployment.Spec.Replicas)

	log.Infof("current scale: %v", curScale)
	log.Infof("query: %v", query)

	scaleUpWhen := deployment.Annotations[DeploymentAnnotationScaleUpWhen]
	scaleDownWhen := deployment.Annotations[DeploymentAnnotationScaleDownWhen]

	log.Infof("scaleUpWhen: %v", scaleUpWhen)
	log.Infof("scaleDownWhen: %v", scaleDownWhen)

	return func(result float64) (int64, error) {

		parameters := make(map[string]interface{}, 1)
		parameters["result"] = result
		exprScaleUpWhen, err := govaluate.NewEvaluableExpression(scaleUpWhen)

		if err != nil {
			return 0, fmt.Errorf("exprScaleUpWhen: %v", err)
		}

		exprScaleDownWhen, err := govaluate.NewEvaluableExpression(scaleDownWhen)

		if err != nil {
			return 0, fmt.Errorf("exprScaleDownWhen: %v", err)
		}

		scaleUp, err := exprScaleUpWhen.Evaluate(parameters)

		if err != nil {
			return 0, fmt.Errorf("exprScaleUpWhen.Evaluate: %v", err)
		}

		scaleDown, err := exprScaleDownWhen.Evaluate(parameters)

		if err != nil {
			return 0, fmt.Errorf("exprScaleDownWhen.Evalute: %v", err)
		}

		// scale up or down
		log.Infof("scaleUp: %v", scaleUp)
		log.Infof("scaleDown: %v", scaleDown)

		newScale := curScale
		if scaleUp == true && newScale < maxScale {
			newScale++
		}
		if scaleDown == true && newScale > minScale {
			newScale--
		}

		return newScale, nil
	}, query, nil
}
