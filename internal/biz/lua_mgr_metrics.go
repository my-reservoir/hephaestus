package biz

import "github.com/prometheus/client_golang/prometheus"

var (
	cntCompiledScripts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hephaestus_compiled_scripts_total",
		Help: "Total number of full compilation of scripts",
	})
	cntFailedCompiledScripts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hephaestus_compilation_failures_total",
		Help: "Total number of compilation failures",
	})
	cntNewedKeys = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "hephaestus_new_keys_total",
		Help: "Total number of newly allocated keys",
	})
)

func init() {
	prometheus.MustRegister(cntCompiledScripts, cntFailedCompiledScripts, cntNewedKeys)
}
