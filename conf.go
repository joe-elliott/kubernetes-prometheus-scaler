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

func (m *MetricConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type mcAlias MetricConfig
	conf := &mcAlias{
		Namespace:            "",
		Name:                 "",
		Dimensions:           nil,
		DimensionSelect:      nil,
		DimensionSelectRegex: nil,
		Statistics:           nil,
		DelaySeconds:         600,
		RangeSeconds:         600,
		PeriodSeconds:        60,
	}

	err := unmarshal(conf)

	*m = MetricConfig(*conf)
	return err
}

func (c *GlobalConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type gcAlias GlobalConfig
	conf := &gcAlias{
		Region:        "",
		Metrics:       nil,
		DelaySeconds:  -1,
		RangeSeconds:  -1,
		PeriodSeconds: -1,
	}

	err := unmarshal(conf)

	*c = GlobalConfig(*conf)
	return err
}
