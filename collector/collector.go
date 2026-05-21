package collector

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	VoltagePortCore   = "core"
	VoltagePortSDRamC = "sdram_c"
	VoltagePortSDRamP = "sdram_p"
	VoltagePortSDRamI = "sdram_i"

	ClockIDARM  = "arm"
	ClockIDCore = "core"

	MemIDARM = "arm"
	MemIDGPU = "gpu"
)

func VoltagePorts() []string {
	return []string{VoltagePortCore, VoltagePortSDRamC, VoltagePortSDRamP, VoltagePortSDRamI}
}

func ClockIDs() []string {
	return []string{ClockIDARM, ClockIDCore, "v3d", "isp", "h264", "pixel", "uart"}
}

func MemIDs() []string { return []string{MemIDARM, MemIDGPU} }

// throttledBit maps a bitmask position to its Prometheus labels.
type throttledBit struct {
	bit       uint
	condition string
	period    string
}

// throttledBits defines the meaning of each relevant bit in the get_throttled bitmask.
// See: https://www.raspberrypi.com/documentation/computers/os.html#get_throttled
var throttledBits = []throttledBit{
	{0, "under_voltage", "now"},
	{1, "freq_capped", "now"},
	{2, "throttled", "now"},
	{3, "soft_temp_limit", "now"},
	{16, "under_voltage", "since_boot"},
	{17, "freq_capped", "since_boot"},
	{18, "throttled", "since_boot"},
	{19, "soft_temp_limit", "since_boot"},
}

var (
	voltageDesc = prometheus.NewDesc(
		"rpi_voltage_volts",
		"Voltage of various Raspberry Pi components in Volts.",
		[]string{"port"}, nil,
	)
	throttledDesc = prometheus.NewDesc(
		"rpi_throttled_status",
		"Raspberry Pi throttled status (bitmask). See vcgencmd get_throttled output.",
		nil, nil,
	)
	throttledDetailDesc = prometheus.NewDesc(
		"rpi_throttled",
		"Raspberry Pi throttled condition (0=ok, 1=active). Labels: condition and period (now|since_boot).",
		[]string{"condition", "period"}, nil,
	)
	temperatureDesc = prometheus.NewDesc(
		"rpi_temperature_celsius",
		"Temperature of the Raspberry Pi SoC in Celsius.",
		nil, nil,
	)
	clockDesc = prometheus.NewDesc(
		"rpi_clock_frequency_hertz",
		"Clock frequency of various Raspberry Pi components in Hertz.",
		[]string{"id"}, nil,
	)
	memoryDesc = prometheus.NewDesc(
		"rpi_memory_bytes",
		"Memory allocated to various Raspberry Pi components in Bytes.",
		[]string{"id"}, nil,
	)
	resetReasonDesc = prometheus.NewDesc(
		"rpi_reset_reason",
		"Raspberry Pi reset reason bitmask (vcgencmd get_rsts).",
		nil, nil,
	)
)

type RPiCollector struct{}

func NewRPiCollector() *RPiCollector { return &RPiCollector{} }

// Describe implements the prometheus.Collector interface.
func (c *RPiCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- voltageDesc
	ch <- throttledDesc
	ch <- throttledDetailDesc
	ch <- temperatureDesc
	ch <- clockDesc
	ch <- memoryDesc
	ch <- resetReasonDesc
}

// Collect implements the prometheus.Collector interface.
func (c *RPiCollector) Collect(ch chan<- prometheus.Metric) {
	for _, port := range VoltagePorts() {
		v, err := GetVoltage(port)
		if err != nil {
			slog.Error("Error collecting voltage", "port", port, "error", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(voltageDesc, prometheus.GaugeValue, v, port)
	}

	if raw, err := GetThrottledStatus(); err != nil {
		slog.Error("Error collecting throttled status", "error", err)
	} else {
		ch <- prometheus.MustNewConstMetric(throttledDesc, prometheus.GaugeValue, raw)
		mask := uint64(raw)
		for _, b := range throttledBits {
			val := float64((mask >> b.bit) & 1)
			ch <- prometheus.MustNewConstMetric(throttledDetailDesc, prometheus.GaugeValue, val, b.condition, b.period)
		}
	}

	if temp, err := GetTemperature(); err != nil {
		slog.Error("Error collecting temperature", "error", err)
	} else {
		ch <- prometheus.MustNewConstMetric(temperatureDesc, prometheus.GaugeValue, temp)
	}

	for _, id := range ClockIDs() {
		freq, err := GetClock(id)
		if err != nil {
			slog.Error("Error collecting clock frequency", "id", id, "error", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(clockDesc, prometheus.GaugeValue, freq, id)
	}

	for _, id := range MemIDs() {
		mem, err := GetMemory(id)
		if err != nil {
			slog.Error("Error collecting memory", "id", id, "error", err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(memoryDesc, prometheus.GaugeValue, mem, id)
	}

	if reason, err := GetResetReason(); err != nil {
		slog.Error("Error collecting reset reason", "error", err)
	} else {
		ch <- prometheus.MustNewConstMetric(resetReasonDesc, prometheus.GaugeValue, reason)
	}
}
