package util

import (
	"testing"

	"github.com/Knetic/govaluate"
)

type testCase struct {
	description   string
	expectedScale int64
	scalable      Scalable
}

var _trueExpr, _ = govaluate.NewEvaluableExpression("true")
var _falseExpr, _ = govaluate.NewEvaluableExpression("false")

var _testCases = []testCase{
	{
		description:   "Will Scale Up",
		expectedScale: 4,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
				query:    "query",
				minScale: 2,
				maxScale: 4,
				curScale: 3,
			},
			scaleUpWhen:   _trueExpr,
			scaleDownWhen: _falseExpr,
		},
	},
	{
		description:   "Will Scale Down",
		expectedScale: 2,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
				query:    "query",
				minScale: 2,
				maxScale: 4,
				curScale: 3,
			},
			scaleUpWhen:   _falseExpr,
			scaleDownWhen: _trueExpr,
		},
	},
	{
		description:   "Stays the same (false)",
		expectedScale: 3,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
				query:    "query",
				minScale: 2,
				maxScale: 4,
				curScale: 3,
			},
			scaleUpWhen:   _falseExpr,
			scaleDownWhen: _falseExpr,
		},
	},
	{
		description:   "Stays the same (true)",
		expectedScale: 3,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
				query:    "query",
				minScale: 2,
				maxScale: 4,
				curScale: 3,
			},
			scaleUpWhen:   _trueExpr,
			scaleDownWhen: _trueExpr,
		},
	},
}

func TestStepScalable(t *testing.T) {

	for _, tc := range _testCases {

		func(tc testCase) {

			t.Run(tc.description, func(t *testing.T) {

				newScale, err := CalculateNewScale(tc.scalable, 0.0)

				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if newScale != tc.expectedScale {
					t.Errorf("Test [%v] failed.", tc.description)
				}

			})

		}(tc)
	}

}
