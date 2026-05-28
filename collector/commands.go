// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package collector

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// systemLock prevents command overload.
// Only one command should be called at time to high load on the system.
var systemLock = &sync.Mutex{}
var isRpi = atomic.Bool{}
var once = sync.Once{}

// best-effort check if we use a pi or not
func IsRpi() bool {
	once.Do(func() {
		_, err := exec.LookPath("vcgencmd")
		status := err == nil
		if status {
			slog.Info("vcgencmd found, storing raspi status", "is_rpi", status)
		} else {
			slog.Warn("vcgencmd not found, storing raspi status", "is_rpi", status, "error", err)
		}
		isRpi.Store(status)
	})
	return isRpi.Load()
}

// runVCGenCmd executes a vcgencmd command with arguments and returns the output.
// It uses the systemLock to prevent concurrent calls.
func runVCGenCmd(args ...string) (string, error) {
	systemLock.Lock()
	defer systemLock.Unlock()

	if !IsRpi() {
		return "", fmt.Errorf("vcgencmd not available (not a Raspberry Pi)")
	}

	cmd := exec.Command("vcgencmd", args...)
	output, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		slog.Error("failed to run vcgencmd", "args", args, "error", err, "stderr", stderr)
		return "", errors.New("vcgencmd error")
	}

	return strings.TrimSpace(string(output)), nil
}

// GetVoltage runs vcgencmd measure_volts for a given port and returns the voltage.
func GetVoltage(port string) (float64, error) {
	output, err := runVCGenCmd("measure_volts", port)
	if err != nil {
		return 0, err
	}

	// Example output: volt=1.2000V
	re := regexp.MustCompile(`volt=(\d+\.?\d*)V`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse voltage from output: %s", output)
	}

	voltageStr := matches[1]
	voltage, err := strconv.ParseFloat(voltageStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from voltage string '%s': %w", voltageStr, err)
	}

	return voltage, nil
}

// GetThrottledStatus runs vcgencmd get_throttled and returns the status as a float64.
func GetThrottledStatus() (float64, error) {
	output, err := runVCGenCmd("get_throttled")
	if err != nil {
		return 0, err
	}

	// Example output: throttled=0x0
	re := regexp.MustCompile(`throttled=(0x[0-9a-fA-F]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse throttled status from output: %s", output)
	}

	statusHex := matches[1]
	// Parse hex string to integer, then convert to float64 for the gauge
	statusInt, err := strconv.ParseInt(statusHex, 0, 64) // 0 infers base from prefix (0x)
	if err != nil {
		return 0, fmt.Errorf("could not parse int from throttled status hex '%s': %w", statusHex, err)
	}

	return float64(statusInt), nil
}

// GetTemperature runs vcgencmd measure_temp and returns the temperature in Celsius.
func GetTemperature() (float64, error) {
	output, err := runVCGenCmd("measure_temp")
	if err != nil {
		return 0, err
	}

	// Example output: temp=45.0'C
	re := regexp.MustCompile(`temp=(\d+\.?\d*)'C`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse temperature from output: %s", output)
	}

	tempStr := matches[1]
	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from temperature string '%s': %w", tempStr, err)
	}

	return temp, nil
}

// GetClock runs vcgencmd measure_clock for a given clock ID and returns the frequency in Hertz.
func GetClock(id string) (float64, error) {
	output, err := runVCGenCmd("measure_clock", id)
	if err != nil {
		return 0, err
	}

	// Example output: frequency(48)=900228544
	re := regexp.MustCompile(`frequency\(\d+\)=(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse clock frequency for %s from output: %s", id, output)
	}

	freqStr := matches[1]
	freq, err := strconv.ParseFloat(freqStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from frequency string '%s': %w", freqStr, err)
	}

	return freq, nil
}

// GetMemory runs vcgencmd get_mem for a given memory ID and returns the memory in Bytes.
func GetMemory(id string) (float64, error) {
	output, err := runVCGenCmd("get_mem", id)
	if err != nil {
		return 0, err
	}

	// Example output: arm=512M
	re := regexp.MustCompile(fmt.Sprintf(`%s=(\d+)M`, regexp.QuoteMeta(id)))
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse memory for %s from output: %s", id, output)
	}

	memStr := matches[1]
	// Memory is reported in MB, convert to Bytes
	memMB, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from memory string '%s': %w", memStr, err)
	}

	return memMB * 1024 * 1024, nil // Convert MB to Bytes
}

// GetCMAFromProcMeminfo reads /proc/meminfo and returns CmaTotal and CmaFree in bytes.
// This reflects kernel CMA (e.g. reserved via dtoverlay=vc4-kms-v3d,cma-512), not VideoCore-managed CMA.
func GetCMAFromProcMeminfo() (reserved, free float64, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read /proc/meminfo: %w", err)
	}
	s := string(data)
	extract := func(key string) (float64, error) {
		re := regexp.MustCompile(key + `:\s+(\d+)\s+kB`)
		m := re.FindStringSubmatch(s)
		if len(m) < 2 {
			return 0, fmt.Errorf("%s not found in /proc/meminfo", key)
		}
		kb, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}

	reserved, err = extract("CmaTotal")
	if err != nil {
		return 0, 0, err
	}
	free, err = extract("CmaFree")
	if err != nil {
		return 0, 0, err
	}
	return reserved, free, nil
}

// v3dBoStatsPath is the debugfs path for V3D GPU buffer object statistics.
const v3dBoStatsPath = "/sys/kernel/debug/dri/0/bo_stats"

// GetV3DBoStats reads the V3D DRM debugfs bo_stats file and returns the count and
// total size in bytes of currently allocated GPU buffer objects.
func GetV3DBoStats() (objects, bytes float64, err error) {
	data, err := os.ReadFile(v3dBoStatsPath)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot read %s: %w", v3dBoStatsPath, err)
	}
	s := string(data)
	extract := func(prefix string) (float64, error) {
		re := regexp.MustCompile(`(?m)^` + prefix + `[^0-9]*(\d+)`)
		m := re.FindStringSubmatch(s)
		if len(m) < 2 {
			return 0, fmt.Errorf("%q not found in %s", prefix, v3dBoStatsPath)
		}
		return strconv.ParseFloat(m[1], 64)
	}

	objects, err = extract("allocated bos")
	if err != nil {
		return 0, 0, err
	}
	kb, err := extract("allocated bo size")
	if err != nil {
		return 0, 0, err
	}
	return objects, kb * 1024, nil
}

// MemRelocStats holds VideoCore relocatable heap statistics from vcgencmd mem_reloc_stats.
type MemRelocStats struct {
	AllocFailures    float64
	Compactions      float64
	LegacyBlockFails float64
}

// GetMemRelocStats runs vcgencmd mem_reloc_stats and returns parsed statistics.
func GetMemRelocStats() (MemRelocStats, error) {
	output, err := runVCGenCmd("mem_reloc_stats")
	if err != nil {
		return MemRelocStats{}, err
	}
	reNum := regexp.MustCompile(`(\d+)`)
	var stats MemRelocStats
	for _, line := range strings.Split(output, "\n") {
		m := reNum.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		val, _ := strconv.ParseFloat(m[1], 64)
		switch {
		case strings.Contains(line, "alloc failures"):
			stats.AllocFailures = val
		case strings.Contains(line, "compactions"):
			stats.Compactions = val
		case strings.Contains(line, "legacy block fails"):
			stats.LegacyBlockFails = val
		}
	}
	return stats, nil
}

// MemOOMStats holds VideoCore OOM statistics from vcgencmd mem_oom.
type MemOOMStats struct {
	Events      float64
	LifetimeMB  float64
	TotalTimeMS float64
	MaxTimeMS   float64
}

// GetMemOOM runs vcgencmd mem_oom and returns parsed statistics.
func GetMemOOM() (MemOOMStats, error) {
	output, err := runVCGenCmd("mem_oom")
	if err != nil {
		return MemOOMStats{}, err
	}
	reNum := regexp.MustCompile(`(\d+)`)
	var stats MemOOMStats
	for _, line := range strings.Split(output, "\n") {
		m := reNum.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		val, _ := strconv.ParseFloat(m[1], 64)
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "oom events"):
			stats.Events = val
		case strings.Contains(line, "lifetime oom required"):
			stats.LifetimeMB = val
		case strings.HasPrefix(strings.TrimSpace(line), "total time"):
			stats.TotalTimeMS = val
		case strings.HasPrefix(strings.TrimSpace(line), "max time"):
			stats.MaxTimeMS = val
		}
	}
	return stats, nil
}

// GetResetReason runs vcgencmd get_rsts and returns the reset reason bitmask.
func GetResetReason() (float64, error) {
	output, err := runVCGenCmd("get_rsts")
	if err != nil {
		return 0, err
	}

	// Example output: get_rsts=1000
	re := regexp.MustCompile(`get_rsts=(\d+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse reset reason from output: %s", output)
	}

	v, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from reset reason '%s': %w", matches[1], err)
	}

	return v, nil
}
