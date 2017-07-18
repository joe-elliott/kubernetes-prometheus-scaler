package util

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
)

func MakePrometheusQueryFunc(url string) (func(query string) (float64, error), error) {

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
