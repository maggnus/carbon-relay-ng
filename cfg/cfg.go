package cfg

import (
	"github.com/graphite-ng/carbon-relay-ng/route"
	"github.com/graphite-ng/carbon-relay-ng/validate"
)

type Config struct {
	Listen_addr             string
	Pickle_addr             string
	Admin_addr              string
	Http_addr               string
	Spool_dir               string
	Max_procs               int
	First_only              bool
	Routes                  []*route.Route
	Init                    []string
	Instance                string
	Log_level               string
	Instrumentation         instrumentation
	Bad_metrics_max_age     string
	Pid_file                string
	Validation_level_legacy validate.LevelLegacy
	Validation_level_m20    validate.LevelM20
	Validate_order          bool
}

type instrumentation struct {
	Graphite_addr     string
	Graphite_interval int
}
