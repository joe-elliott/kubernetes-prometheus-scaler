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

var _fiveExpr, _ = govaluate.NewEvaluableExpression("5")

var _testCases = []testCase{
	// step scalable
	{
		description:   "Will Scale Up",
		expectedScale: 4,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
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
				minScale: 2,
				maxScale: 4,
				curScale: 3,
			},
			scaleUpWhen:   _trueExpr,
			scaleDownWhen: _trueExpr,
		},
	},
	{
		description:   "Won't scale past max",
		expectedScale: 4,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
				minScale: 2,
				maxScale: 4,
				curScale: 4,
			},
			scaleUpWhen:   _trueExpr,
			scaleDownWhen: _falseExpr,
		},
	},
	{
		description:   "Won't scale below min",
		expectedScale: 2,
		scalable: &StepScalable{
			BaseScalable: BaseScalable{
				minScale: 2,
				maxScale: 4,
				curScale: 2,
			},
			scaleUpWhen:   _falseExpr,
			scaleDownWhen: _trueExpr,
		},
	},
	// direct scalable
	{
		description:   "Scales directly",
		expectedScale: 5,
		scalable: &DirectScalable{
			BaseScalable: BaseScalable{
				minScale: 4,
				maxScale: 6,
			},
			scaleTo: _fiveExpr,
		},
	},
	{
		description:   "Scaling directly won't go below min",
		expectedScale: 6,
		scalable: &DirectScalable{
			BaseScalable: BaseScalable{
				minScale: 6,
				maxScale: 9,
			},
			scaleTo: _fiveExpr,
		},
	},
	{
		description:   "Scaling directly won't go over max",
		expectedScale: 4,
		scalable: &DirectScalable{
			BaseScalable: BaseScalable{
				minScale: 2,
				maxScale: 4,
			},
			scaleTo: _fiveExpr,
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
