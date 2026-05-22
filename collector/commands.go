// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package collector

import (
	"errors"
	"fmt"
	"log/slog"
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

var (
	reVolt      = regexp.MustCompile(`volt=(\d+\.?\d*)V`)
	reThrottled = regexp.MustCompile(`throttled=(0x[0-9a-fA-F]+)`)
	reTemp      = regexp.MustCompile(`temp=(\d+\.?\d*)'C`)
	reClock     = regexp.MustCompile(`frequency\(\d+\)=(\d+)`)
	reMem       = regexp.MustCompile(`([^=]+)=(\d+)M`)
	reRsts      = regexp.MustCompile(`get_rsts=(\d+)`)
)

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
		slog.Error("failed to run vcgencmd", "args", args, "error", err)
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
	matches := reVolt.FindStringSubmatch(output)
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
	matches := reThrottled.FindStringSubmatch(output)
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
	matches := reTemp.FindStringSubmatch(output)
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
	matches := reClock.FindStringSubmatch(output)
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
	matches := reMem.FindStringSubmatch(output)
	if len(matches) < 3 || matches[1] != id {
		return 0, fmt.Errorf("could not parse memory for %s from output: %s", id, output)
	}

	memStr := matches[2]
	// Memory is reported in MB, convert to Bytes
	memMB, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from memory string '%s': %w", memStr, err)
	}

	return memMB * 1024 * 1024, nil // Convert MB to Bytes
}

// GetResetReason runs vcgencmd get_rsts and returns the reset reason bitmask.
func GetResetReason() (float64, error) {
	output, err := runVCGenCmd("get_rsts")
	if err != nil {
		return 0, err
	}

	// Example output: get_rsts=1000
	matches := reRsts.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse reset reason from output: %s", output)
	}

	v, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse float from reset reason '%s': %w", matches[1], err)
	}

	return v, nil
}
