package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type GlobalConfig struct {
	Region        string
	Metrics       []MetricConfig
	DelaySeconds  int `yaml:"delay_seconds"`
	RangeSeconds  int `yaml:"range_seconds"`
	PeriodSeconds int `yaml:"period_seconds"`
}

type MetricConfig struct {
	Namespace            string
	Name                 string
	Dimensions           []string
	DimensionSelect      map[string]string `yaml:"dimension_select"`
	DimensionSelectRegex map[string]string `yaml:"dimension_select_regex"`
	Statistics           []string
	DelaySeconds         int `yaml:"delay_seconds"`
	RangeSeconds         int `yaml:"range_seconds"`
	PeriodSeconds        int `yaml:"period_seconds"`
}

func loadConf(file string) (GlobalConfig, error) {

	var conf GlobalConfig

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

/*
// custom marshalling to set defaults
func (m *MetricConfig) UnmarshalYAML(data []byte) error {
	type mcAlias MetricConfig
	conf := &mcAlias{
		Namespace:            "",
		Name:                 "",
		Dimensions:           nil,
		DimensionSelect:      nil,
		DimensionSelectRegex: nil,
		Statistics:           nil,
		DelaySeconds:         -1,
		RangeSeconds:         -1,
		PeriodSeconds:        -1,
	}

	_ = json.Unmarshal(data, conf)

	*m = MetricConfig(*conf)
	return nil
}
*/
