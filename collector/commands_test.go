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

	// Test GetMemory for all IDs; VideoCore CMA values may be 0 if not VC-managed.
	zeroable := map[string]bool{collector.MemIDVCCMA: true, collector.MemIDVCCMATotal: true}
	for _, id := range collector.MemIDs() {
		t.Run("Memory_"+id, func(t *testing.T) {
			mem, err := collector.GetMemory(id)
			require.NoError(t, err)
			if zeroable[id] {
				require.GreaterOrEqual(t, mem, 0.0)
			} else {
				require.Greater(t, mem, 0.0)
			}
		})
	}

	t.Run("CMAFromProcMeminfo", func(t *testing.T) {
		reserved, free, err := collector.GetCMAFromProcMeminfo()
		require.NoError(t, err)
		require.Greater(t, reserved, 0.0)
		require.GreaterOrEqual(t, free, 0.0)
	})

	t.Run("MemRelocStats", func(t *testing.T) {
		stats, err := collector.GetMemRelocStats()
		require.NoError(t, err)
		require.GreaterOrEqual(t, stats.AllocFailures, 0.0)
		require.GreaterOrEqual(t, stats.Compactions, 0.0)
		require.GreaterOrEqual(t, stats.LegacyBlockFails, 0.0)
	})

	t.Run("V3DBoStats", func(t *testing.T) {
		objects, bytes, err := collector.GetV3DBoStats()
		require.NoError(t, err)
		require.GreaterOrEqual(t, objects, 0.0)
		require.GreaterOrEqual(t, bytes, 0.0)
	})

	t.Run("MemOOM", func(t *testing.T) {
		stats, err := collector.GetMemOOM()
		require.NoError(t, err)
		require.GreaterOrEqual(t, stats.Events, 0.0)
		require.GreaterOrEqual(t, stats.LifetimeMB, 0.0)
		require.GreaterOrEqual(t, stats.TotalTimeMS, 0.0)
		require.GreaterOrEqual(t, stats.MaxTimeMS, 0.0)
	})
}

func TestIsRpi(t *testing.T) {
	v := collector.IsRpi()
	t.Log("IsRpi:", v)
}
