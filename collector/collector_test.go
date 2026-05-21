// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package collector_test

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"ubunatic.com/rpi-exporter/collector"
)

func gatherMetrics(t *testing.T) map[string]*dto.MetricFamily {
	t.Helper()
	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewRPiCollector())

	mfs, err := reg.Gather()
	require.NoError(t, err)

	families := make(map[string]*dto.MetricFamily, len(mfs))
	for _, mf := range mfs {
		families[mf.GetName()] = mf
	}
	return families
}

func labelValues(metrics []*dto.Metric, label string) []string {
	vals := make([]string, 0, len(metrics))
	for _, m := range metrics {
		for _, lp := range m.GetLabel() {
			if lp.GetName() == label {
				vals = append(vals, lp.GetValue())
			}
		}
	}
	return vals
}

func TestCollector_AllFamiliesPresent(t *testing.T) {
	families := gatherMetrics(t)

	expected := []string{
		"rpi_voltage_volts",
		"rpi_throttled_status",
		"rpi_temperature_celsius",
		"rpi_clock_frequency_hertz",
		"rpi_memory_bytes",
	}
	for _, name := range expected {
		require.Contains(t, families, name, "missing metric family: %s", name)
	}
}

func TestCollector_Voltage(t *testing.T) {
	families := gatherMetrics(t)

	metrics := families["rpi_voltage_volts"].GetMetric()
	require.Len(t, metrics, len(collector.VoltagePorts()))

	ports := labelValues(metrics, "port")
	require.ElementsMatch(t, collector.VoltagePorts(), ports)

	for _, m := range metrics {
		require.Greater(t, m.GetGauge().GetValue(), 0.0, "voltage must be > 0")
	}
}

func TestCollector_ThrottledStatus(t *testing.T) {
	families := gatherMetrics(t)

	metrics := families["rpi_throttled_status"].GetMetric()
	require.Len(t, metrics, 1)
	require.GreaterOrEqual(t, metrics[0].GetGauge().GetValue(), 0.0)
}

func TestCollector_Temperature(t *testing.T) {
	families := gatherMetrics(t)

	metrics := families["rpi_temperature_celsius"].GetMetric()
	require.Len(t, metrics, 1)
	require.GreaterOrEqual(t, metrics[0].GetGauge().GetValue(), 0.0)
}

func TestCollector_Clock(t *testing.T) {
	families := gatherMetrics(t)

	metrics := families["rpi_clock_frequency_hertz"].GetMetric()
	require.Len(t, metrics, len(collector.ClockIDs()))

	ids := labelValues(metrics, "id")
	require.ElementsMatch(t, collector.ClockIDs(), ids)

	for _, m := range metrics {
		require.Greater(t, m.GetGauge().GetValue(), 0.0, "clock frequency must be > 0")
	}
}

func TestCollector_Memory(t *testing.T) {
	families := gatherMetrics(t)

	metrics := families["rpi_memory_bytes"].GetMetric()
	require.Len(t, metrics, len(collector.MemIDs()))

	ids := labelValues(metrics, "id")
	require.ElementsMatch(t, collector.MemIDs(), ids)

	for _, m := range metrics {
		require.Greater(t, m.GetGauge().GetValue(), 0.0, "memory must be > 0")
	}
}
