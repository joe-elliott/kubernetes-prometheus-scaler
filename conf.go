package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type globalConfig struct {
	region        string
	metrics       []metricConfig
	delaySeconds  int // 600
	rangeSeconds  int // 600
	periodSeconds int // 60
}

type metricConfig struct {
	namespace            string
	name                 string
	dimensions           []string
	dimensionSelect      map[string]string
	dimensionSelectRegex map[string]string
	statistics           []string
	delaySeconds         int
	rangeSeconds         int
	periodSeconds        int
}

func loadConf(file string) (globalConfig, error) {

	conf := globalConfig{}

	data, err := ioutil.ReadFile(file)

	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(data, &conf)

	if err != nil {
		return conf, err
	}

	return conf, nil
}
