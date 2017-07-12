package scaler

import (
	"errors"
	"fmt"
	"math"
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
const DeploymentAnnotationScaleTo = "prometheusScaler/scale-to"

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

type DirectScalable struct {
	BaseScalable
	scaleTo *govaluate.EvaluableExpression
}

func NewScalable(deployment v1beta1.Deployment) (Scalable, error) {
	var err error

	// parse scaling parameters from deployment spec
	query := deployment.Annotations[DeploymentAnnotationPrometheusQuery]
	minScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMinScale], 10, 32)

	if err != nil {
		return nil, fmt.Errorf("minScale: %v", err)
	}

	maxScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMaxScale], 10, 32)

	if err != nil {
		return nil, fmt.Errorf("maxScale: %v", err)
	}

	// get current state
	curScale := int64(*deployment.Spec.Replicas)

	scaleUpWhen := deployment.Annotations[DeploymentAnnotationScaleUpWhen]
	scaleDownWhen := deployment.Annotations[DeploymentAnnotationScaleDownWhen]
	scaleTo := deployment.Annotations[DeploymentAnnotationScaleTo]

	log.Infof("scaleUpWhen: %v", scaleUpWhen)
	log.Infof("scaleDownWhen: %v", scaleDownWhen)
	log.Infof("scaleTo: %v", scaleTo)

	if scaleUpWhen != "" && scaleDownWhen != "" {
		scalable := StepScalable{}

		scalable.scaleUpWhen, err = govaluate.NewEvaluableExpression(scaleUpWhen)

		if err != nil {
			return nil, fmt.Errorf("exprScaleUpWhen: %v", err)
		}

		scalable.scaleDownWhen, err = govaluate.NewEvaluableExpression(scaleDownWhen)

		if err != nil {
			return nil, fmt.Errorf("exprScaleDownWhen: %v", err)
		}

		// oh lordy there has to be a better way
		//  where is easy mode polymorphism?
		scalable.query = query
		scalable.curScale = curScale
		scalable.maxScale = maxScale
		scalable.minScale = minScale

		return scalable, nil
	} else if scaleTo != "" {
		scalable := DirectScalable{}

		scalable.scaleTo, err = govaluate.NewEvaluableExpression(scaleTo)

		if err != nil {
			return nil, fmt.Errorf("exprScaleTo: %v", err)
		}

		// oh lordy there has to be a better way
		//  where is easy mode polymorphism?
		scalable.query = query
		scalable.curScale = curScale
		scalable.maxScale = maxScale
		scalable.minScale = minScale

		return scalable, nil
	}

	return nil, errors.New("Deployment needs either scaleUp/scaleDown annotations or a scaleTo annotation")
}

func MakeScalingFunc(scalable Scalable) (func(result float64) (int64, error), error) {

	switch s := scalable.(type) {
	case StepScalable:
		return func(result float64) (int64, error) {

			parameters := make(map[string]interface{}, 1)
			parameters["result"] = result

			scaleUp, err := s.scaleUpWhen.Evaluate(parameters)

			if err != nil {
				return 0, fmt.Errorf("exprScaleUpWhen.Evaluate: %v", err)
			}

			scaleDown, err := s.scaleDownWhen.Evaluate(parameters)

			if err != nil {
				return 0, fmt.Errorf("exprScaleDownWhen.Evaluate: %v", err)
			}

			// scale up or down
			log.Infof("scaleUp: %v", scaleUp)
			log.Infof("scaleDown: %v", scaleDown)

			newScale := s.curScale
			if scaleUp == true && newScale < s.maxScale {
				newScale++
			}
			if scaleDown == true && newScale > s.minScale {
				newScale--
			}

			return newScale, nil
		}, nil
	case DirectScalable:
		return func(result float64) (int64, error) {

			parameters := make(map[string]interface{}, 1)
			parameters["result"] = result

			scaleTo, err := s.scaleTo.Evaluate(parameters)

			if err != nil {
				return 0, fmt.Errorf("exprScaleTo.Evaluate: %v", err)
			}

			// scale up or down
			log.Infof("scaleTo: %v", scaleTo)

			var newScale int64

			if f, ok := scaleTo.(float64); ok {
				// dangerous due to float representation. where's my round() function go?
				newScale = int64(math.Floor(f + .5))

				if newScale > s.maxScale {
					newScale = s.maxScale
				}
				if newScale < s.minScale {
					newScale = s.minScale
				}
			} else {
				return 0, fmt.Errorf("Can't cast %v to int64", scaleTo)
			}

			return newScale, nil
		}, nil
	default:
		return nil, fmt.Errorf("Scalable is an unknown type: %v", scalable)
	}

}
