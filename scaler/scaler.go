package scaler

import (
	"fmt"
	"strconv"

	logging "github.com/op/go-logging"

	"github.com/Knetic/govaluate"
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
	scaleUpWhen   *govaluate.EvaluableExpression
	scaleDownWhen *govaluate.EvaluableExpression
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

	scaleUpWhen := deployment.Annotations[DeploymentAnnotationScaleUpWhen]
	scaleDownWhen := deployment.Annotations[DeploymentAnnotationScaleDownWhen]

	log.Infof("scaleUpWhen: %v", scaleUpWhen)
	log.Infof("scaleDownWhen: %v", scaleDownWhen)

	scalable.scaleUpWhen, err = govaluate.NewEvaluableExpression(scaleUpWhen)

	if err != nil {
		return nil, fmt.Errorf("exprScaleUpWhen: %v", err)
	}

	scalable.scaleDownWhen, err = govaluate.NewEvaluableExpression(scaleDownWhen)

	if err != nil {
		return nil, fmt.Errorf("exprScaleDownWhen: %v", err)
	}

	return scalable, nil
}

func MakeScalingFunc(scalable Scalable) (func(result float64) (int64, error), error) {

	if step, ok := scalable.(StepScalable); ok {

		return func(result float64) (int64, error) {

			parameters := make(map[string]interface{}, 1)
			parameters["result"] = result

			scaleUp, err := step.scaleUpWhen.Evaluate(parameters)

			if err != nil {
				return 0, fmt.Errorf("exprScaleUpWhen.Evaluate: %v", err)
			}

			scaleDown, err := step.scaleDownWhen.Evaluate(parameters)

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
