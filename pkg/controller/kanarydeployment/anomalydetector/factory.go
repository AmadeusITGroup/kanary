package anomalydetector

import (
	"fmt"
)

//FactoryConfig parameters extended with factory features
type FactoryConfig struct {
	Config
	DiscreteValueOutOfListConfig   *DiscreteValueOutOfListConfig
	ContinuousValueDeviationConfig *ContinuousValueDeviationConfig
	ValueInRangeConfig             *ValueInRangeConfig
	PromConfig                     *ConfigPrometheusAnomalyDetector
	CustomService                  string
	customFactory                  Factory //for test purpose
}

//Factory functor for AnomalyDetection
type Factory func(cfg FactoryConfig) (AnomalyDetector, error)

var _ Factory = New

//New Factory for AnomalyDetection
func New(cfg FactoryConfig) (AnomalyDetector, error) {

	errMulti := fmt.Errorf("invalide multiple configuration")
	if cfg.CustomService != "" {
		if cfg.DiscreteValueOutOfListConfig != nil || cfg.ContinuousValueDeviationConfig != nil || cfg.ValueInRangeConfig != nil {
			return nil, errMulti
		}
	}
	if cfg.DiscreteValueOutOfListConfig != nil {
		if cfg.ContinuousValueDeviationConfig != nil || cfg.ValueInRangeConfig != nil {
			return nil, errMulti
		}
	}
	if cfg.ContinuousValueDeviationConfig != nil {
		if cfg.DiscreteValueOutOfListConfig != nil || cfg.ValueInRangeConfig != nil {
			return nil, errMulti
		}
	}

	switch {
	case cfg.PromConfig != nil && cfg.DiscreteValueOutOfListConfig != nil:
		cfg.PromConfig.logger = cfg.Logger
		return newDiscreteValueOutOfListWithProm(cfg.Config, *cfg.DiscreteValueOutOfListConfig, *cfg.PromConfig)
	case cfg.PromConfig != nil && cfg.ContinuousValueDeviationConfig != nil:
		cfg.PromConfig.logger = cfg.Logger
		return newContinuousValueDeviationWithProm(cfg.Config, *cfg.ContinuousValueDeviationConfig, *cfg.PromConfig)
	case cfg.PromConfig != nil && cfg.ValueInRangeConfig != nil:
		cfg.PromConfig.logger = cfg.Logger
		return newValueInRangeWithProm(cfg.Config, *cfg.ValueInRangeConfig, *cfg.PromConfig)
	case cfg.CustomService != "":
		return newCustomAnalyser(cfg.CustomService, cfg.Config)
	case cfg.customFactory != nil:
		return cfg.customFactory(cfg)
	default:
		return nil, fmt.Errorf("no anomaly detection could be built, missing or incomplete configuration")
	}
}

func newCustomAnalyser(customService string, cfg Config) (*CustomAnomalyDetector, error) {
	c := &CustomAnomalyDetector{
		serviceURI: customService,
		logger:     cfg.Logger,
	}
	c.init()
	return c, nil
}

//newValueInRangeWithProm buld an anomaly detector for value in range based on prometheus
func newValueInRangeWithProm(configAnalyser Config, configValueInRange ValueInRangeConfig, configProm ConfigPrometheusAnomalyDetector) (AnomalyDetector, error) {

	a := &ValueInRangeAnalyser{
		ConfigAnalyser: configAnalyser,
		ConfigSpecific: configValueInRange,
	}

	var err error
	if a.analyser, err = newPromValueInRangeAnalyser(configProm, configValueInRange); err != nil {
		return nil, err
	}
	return a, nil
}

//newContinuousValueDeviationWithProm buld an anomaly detector for Continuous value deviation based on prometheus
func newContinuousValueDeviationWithProm(configAnalyser Config, configContinuousValueDeviation ContinuousValueDeviationConfig, configProm ConfigPrometheusAnomalyDetector) (AnomalyDetector, error) {

	a := &ContinuousValueDeviationAnalyser{
		ConfigAnalyser: configAnalyser,
		ConfigSpecific: configContinuousValueDeviation,
	}

	var err error
	if a.analyser, err = newPromContinuousValueDeviationAnalyser(configProm, configContinuousValueDeviation); err != nil {
		return nil, err
	}
	return a, nil
}

//newDiscreteValueOutOfListWithProm build an anomaly detector for Discrete Value count based on prometheus
func newDiscreteValueOutOfListWithProm(configAnalyser Config, configDiscreteValueOutOfList DiscreteValueOutOfListConfig, configProm ConfigPrometheusAnomalyDetector) (AnomalyDetector, error) {

	a := &DiscreteValueOutOfListAnalyser{
		ConfigAnalyser: configAnalyser,
		ConfigSpecific: configDiscreteValueOutOfList,
	}

	var err error
	if a.analyser, err = newPromDiscreteValueOutOfListAnalyser(configProm, configDiscreteValueOutOfList); err != nil {
		return nil, err
	}
	return a, nil
}
