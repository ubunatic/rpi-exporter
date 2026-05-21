// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"ubunatic.com/rpi-exporter/collector"
)

func TestCommands(t *testing.T) {
	v, err := collector.GetThrottledStatus()
	require.NoError(t, err)
	require.GreaterOrEqual(t, v, 0.0)

	for _, port := range collector.VoltagePorts() {
		t.Run(port, func(t *testing.T) {
			v, err = collector.GetVoltage(port)
			require.NoError(t, err)
			require.Greater(t, v, 0.0)
		})
	}

	// Test GetTemperature
	t.Run("Temperature", func(t *testing.T) {
		temp, err := collector.GetTemperature()
		require.NoError(t, err)
		require.Greater(t, temp, 0.0) // Temperature should be non-negative
	})

	// Test GetClock for all IDs
	for _, id := range collector.ClockIDs() {
		t.Run("Clock_"+id, func(t *testing.T) {
			freq, err := collector.GetClock(id)
			require.NoError(t, err)
			require.Greater(t, freq, 0.0) // Frequency should be non-negative
		})
	}

	// Test GetMemory for all IDs
	for _, id := range collector.MemIDs() {
		t.Run("Memory_"+id, func(t *testing.T) {
			mem, err := collector.GetMemory(id)
			require.NoError(t, err)
			require.Greater(t, mem, 0.0) // Memory should be non-negative
		})
	}
}

func TestIsRpi(t *testing.T) {
	v := collector.IsRpi()
	t.Log("IsRpi:", v)
}
