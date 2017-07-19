package util

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
	GetCurScale() int64

	setQuery(string)
	setCurScale(int64)
	setMinScale(int64)
	setMaxScale(int64)
}

type BaseScalable struct {
	query    string
	minScale int64
	maxScale int64
	curScale int64
}

func (s *BaseScalable) GetQuery() string {
	return s.query
}

func (s *BaseScalable) GetCurScale() int64 {
	return s.curScale
}

func (s *BaseScalable) setQuery(q string) {
	s.query = q
}

func (s *BaseScalable) setMinScale(n int64) {
	s.minScale = n
}

func (s *BaseScalable) setMaxScale(n int64) {
	s.maxScale = n
}

func (s *BaseScalable) setCurScale(n int64) {
	s.curScale = n
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
	var ret Scalable

	// first parse annotations to determine scalable type
	scaleUpWhen := deployment.Annotations[DeploymentAnnotationScaleUpWhen]
	scaleDownWhen := deployment.Annotations[DeploymentAnnotationScaleDownWhen]
	scaleTo := deployment.Annotations[DeploymentAnnotationScaleTo]

	log.Debugf("scaleUpWhen: %v", scaleUpWhen)
	log.Debugf("scaleDownWhen: %v", scaleDownWhen)
	log.Debugf("scaleTo: %v", scaleTo)

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

		ret = &scalable
	} else if scaleTo != "" {
		scalable := DirectScalable{}

		scalable.scaleTo, err = govaluate.NewEvaluableExpression(scaleTo)

		if err != nil {
			return nil, fmt.Errorf("exprScaleTo: %v", err)
		}

		ret = &scalable
	}

	// parse standard fields if we successfully created a scalable
	if ret != nil {

		query := deployment.Annotations[DeploymentAnnotationPrometheusQuery]
		minScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMinScale], 10, 32)

		if err != nil {
			return nil, fmt.Errorf("minScale: %v", err)
		}

		maxScale, err := strconv.ParseInt(deployment.Annotations[DeploymentAnnotationMaxScale], 10, 32)

		if err != nil {
			return nil, fmt.Errorf("maxScale: %v", err)
		}

		curScale := int64(*deployment.Spec.Replicas)

		ret.setQuery(query)
		ret.setCurScale(curScale)
		ret.setMinScale(minScale)
		ret.setMaxScale(maxScale)

		return ret, nil
	}

	return nil, errors.New("Deployment needs either scaleUp and scaleDown annotations or a scaleTo annotation")
}

func CalculateNewScale(scalable Scalable, result float64) (int64, error) {

	parameters := make(map[string]interface{}, 1)
	parameters["result"] = result

	switch s := scalable.(type) {
	case *StepScalable:
		return calculateStepScale(s, result, parameters)
	case *DirectScalable:
		return calculateDirectScale(s, result, parameters)
	}

	return 0, fmt.Errorf("Scalable is an unknown type: %v", scalable)
}

func calculateStepScale(s *StepScalable, result float64, parameters map[string]interface{}) (int64, error) {
	scaleUp, err := s.scaleUpWhen.Evaluate(parameters)

	if err != nil {
		return 0, fmt.Errorf("exprScaleUpWhen.Evaluate: %v", err)
	}

	scaleDown, err := s.scaleDownWhen.Evaluate(parameters)

	if err != nil {
		return 0, fmt.Errorf("exprScaleDownWhen.Evaluate: %v", err)
	}

	// scale up or down
	log.Debugf("scaleUp: %v", scaleUp)
	log.Debugf("scaleDown: %v", scaleDown)

	newScale := s.curScale
	if scaleUp == true && newScale < s.maxScale {
		newScale++
	}
	if scaleDown == true && newScale > s.minScale {
		newScale--
	}

	return newScale, nil
}

func calculateDirectScale(s *DirectScalable, result float64, parameters map[string]interface{}) (int64, error) {
	scaleTo, err := s.scaleTo.Evaluate(parameters)

	if err != nil {
		return 0, fmt.Errorf("exprScaleTo.Evaluate: %v", err)
	}

	// scale up or down
	log.Debugf("scaleTo: %v", scaleTo)

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
}
