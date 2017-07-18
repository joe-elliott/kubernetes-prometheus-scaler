package util

import (
	"testing"

	"github.com/Knetic/govaluate"
)

func TestStepScalable(t *testing.T) {

	trueExpr, _ := govaluate.NewEvaluableExpression("true")
	falseExpr, _ := govaluate.NewEvaluableExpression("false")

	s := StepScalable{
		BaseScalable: BaseScalable{
			query:    "query",
			minScale: 2,
			maxScale: 5,
			curScale: 2,
		},
		scaleUpWhen:   trueExpr,
		scaleDownWhen: falseExpr,
	}

	t.Run("Will Scale Up", func(t *testing.T) {
		newScale, err := CalculateNewScale(&s, 0.0)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if newScale != s.curScale+1 {
			t.Error("Did not scale up.")
		}
	})
}
