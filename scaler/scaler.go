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

type Scalable interface {
	GetQuery() string
}

type BaseScalable struct {
	query    string
	minScale int64
	maxScale int64
	curScale int64
}

func (s BaseScalable) GetQuery() string {
	return s.query
}

type StepScalable struct {
	BaseScalable
	scaleUpWhen   string
	scaleDownWhen string
}

func NewScalable(deployment v1beta1.Deployment) (Scalable, error) {
	scalable := StepScalable{}
	var err error

	// parse scaling parameters from deployment spec
	scalable.query = deployment.Annotations[DeploymentAnnotationPrometheusQuery]
	scalable.minScale, err = strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMinScale], 10, 32)

	if err != nil {
		return nil, fmt.Errorf("minScale: %v", err)
	}

	scalable.maxScale, err = strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMaxScale], 10, 32)

	if err != nil {
		return nil, fmt.Errorf("maxScale: %v", err)
	}

	// get current state
	scalable.curScale = int64(*deployment.Spec.Replicas)

	scalable.scaleUpWhen = deployment.Annotations[DeploymentAnnotationScaleUpWhen]
	scalable.scaleDownWhen = deployment.Annotations[DeploymentAnnotationScaleDownWhen]

	return scalable, nil
}

func MakeScalingFunc(scalable Scalable) (func(result float64) (int64, error), error) {

	if step, ok := scalable.(StepScalable); ok {

		return func(result float64) (int64, error) {

			parameters := make(map[string]interface{}, 1)
			parameters["result"] = result
			exprScaleUpWhen, err := govaluate.NewEvaluableExpression(step.scaleUpWhen)

			if err != nil {
				return 0, fmt.Errorf("exprScaleUpWhen: %v", err)
			}

			exprScaleDownWhen, err := govaluate.NewEvaluableExpression(step.scaleDownWhen)

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

			newScale := step.curScale
			if scaleUp == true && newScale < step.maxScale {
				newScale++
			}
			if scaleDown == true && newScale > step.minScale {
				newScale--
			}

			return newScale, nil
		}, nil
	}

	return nil, fmt.Errorf("Scalable is an unknown type: %v", scalable)
}
