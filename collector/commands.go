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

// parseVCGenCmdFloat executes a vcgencmd command, matches output with regex, and parses it.
func parseVCGenCmdFloat(metricName string, re *regexp.Regexp, parseFunc func(string) (float64, error), args ...string) (float64, error) {
	output, err := runVCGenCmd(args...)
	if err != nil {
		return 0, err
	}

	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("could not parse %s from output: %s", metricName, output)
	}

	val, err := parseFunc(matches[1])
	if err != nil {
		return 0, fmt.Errorf("could not parse %s value '%s': %w", metricName, matches[1], err)
	}

	return val, nil
}

// defaultParseFloat parses a string to a float64
func defaultParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

var (
	voltageRe     = regexp.MustCompile(`volt=(\d+\.?\d*)V`)
	throttledRe   = regexp.MustCompile(`throttled=(0x[0-9a-fA-F]+)`)
	temperatureRe = regexp.MustCompile(`temp=(\d+\.?\d*)'C`)
	clockRe       = regexp.MustCompile(`frequency\(\d+\)=(\d+)`)
	resetReasonRe = regexp.MustCompile(`get_rsts=(\d+)`)
)

// GetVoltage runs vcgencmd measure_volts for a given port and returns the voltage.
func GetVoltage(port string) (float64, error) {
	return parseVCGenCmdFloat("voltage", voltageRe, defaultParseFloat, "measure_volts", port)
}

// GetThrottledStatus runs vcgencmd get_throttled and returns the status as a float64.
func GetThrottledStatus() (float64, error) {
	return parseVCGenCmdFloat("throttled status", throttledRe, func(s string) (float64, error) {
		statusInt, err := strconv.ParseInt(s, 0, 64)
		return float64(statusInt), err
	}, "get_throttled")
}

// GetTemperature runs vcgencmd measure_temp and returns the temperature in Celsius.
func GetTemperature() (float64, error) {
	return parseVCGenCmdFloat("temperature", temperatureRe, defaultParseFloat, "measure_temp")
}

// GetClock runs vcgencmd measure_clock for a given clock ID and returns the frequency in Hertz.
func GetClock(id string) (float64, error) {
	return parseVCGenCmdFloat(fmt.Sprintf("clock frequency for %s", id), clockRe, defaultParseFloat, "measure_clock", id)
}

// GetMemory runs vcgencmd get_mem for a given memory ID and returns the memory in Bytes.
func GetMemory(id string) (float64, error) {
	re := regexp.MustCompile(fmt.Sprintf(`%s=(\d+)M`, regexp.QuoteMeta(id)))
	return parseVCGenCmdFloat(fmt.Sprintf("memory for %s", id), re, func(s string) (float64, error) {
		memMB, err := strconv.ParseFloat(s, 64)
		return memMB * 1024 * 1024, err
	}, "get_mem", id)
}

// GetResetReason runs vcgencmd get_rsts and returns the reset reason bitmask.
func GetResetReason() (float64, error) {
	return parseVCGenCmdFloat("reset reason", resetReasonRe, defaultParseFloat, "get_rsts")
}
