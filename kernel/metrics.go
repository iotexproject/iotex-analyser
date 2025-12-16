package kernel

import "github.com/prometheus/client_golang/prometheus"

var (
	ProcessTimeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "iotex_analyser_plugin_inner_processing_seconds_per_block",
			Help:       "iotex analyser plugin inner processing seconds per block",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name", "step"},
	)
)

func init() {
	prometheus.MustRegister(ProcessTimeMetric)
}
