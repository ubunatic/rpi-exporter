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
		"rpi_gpu_reloc_total",
		"rpi_gpu_oom_events_total",
		"rpi_gpu_oom_lifetime_bytes",
		"rpi_gpu_oom_handler_seconds_total",
		"rpi_gpu_oom_handler_max_seconds",
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
	require.Len(t, metrics, len(collector.AllMemIDs()))

	ids := labelValues(metrics, "id")
	require.ElementsMatch(t, collector.AllMemIDs(), ids)

	// VideoCore CMA may be 0 when not VC-managed; all others must be positive.
	zeroable := map[string]bool{collector.MemIDVCCMA: true, collector.MemIDVCCMATotal: true}
	for _, m := range metrics {
		val := m.GetGauge().GetValue()
		id := ""
		for _, lp := range m.GetLabel() {
			if lp.GetName() == "id" {
				id = lp.GetValue()
			}
		}
		if zeroable[id] {
			require.GreaterOrEqual(t, val, 0.0, "memory id=%s must be >= 0", id)
		} else {
			require.Greater(t, val, 0.0, "memory id=%s must be > 0", id)
		}
	}
}

func TestCollector_GPUReloc(t *testing.T) {
	families := gatherMetrics(t)

	metrics := families["rpi_gpu_reloc_total"].GetMetric()
	require.Len(t, metrics, 3)

	events := labelValues(metrics, "event")
	require.ElementsMatch(t, []string{"alloc_failures", "compactions", "legacy_block_fails"}, events)
}

func TestCollector_GPUOOM(t *testing.T) {
	families := gatherMetrics(t)

	for _, name := range []string{
		"rpi_gpu_oom_events_total",
		"rpi_gpu_oom_lifetime_bytes",
		"rpi_gpu_oom_handler_seconds_total",
		"rpi_gpu_oom_handler_max_seconds",
	} {
		m := families[name].GetMetric()
		require.Len(t, m, 1, "expected 1 metric for %s", name)
		require.GreaterOrEqual(t, m[0].GetGauge().GetValue()+m[0].GetCounter().GetValue(), 0.0)
	}
}
