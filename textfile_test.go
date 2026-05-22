// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package rpiexporter

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGatherer implements prometheus.Gatherer for testing
type mockGatherer struct {
	mfs []*dto.MetricFamily
	err error
}

func (m *mockGatherer) Gather() ([]*dto.MetricFamily, error) {
	return m.mfs, m.err
}

func TestWriteTextfile(t *testing.T) {
	// Dummy metrics for more complex edge case testing
	metricName := "test_metric"
	metricHelp := "test metric help"
	metricType := dto.MetricType_GAUGE
	metricValue := float64(42)

	mfs := []*dto.MetricFamily{
		{
			Name: &metricName,
			Help: &metricHelp,
			Type: &metricType,
			Metric: []*dto.Metric{
				{
					Gauge: &dto.Gauge{
						Value: &metricValue,
					},
				},
			},
		},
	}

	t.Run("success_file_real_registry", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "test.prom")

		registry := prometheus.NewRegistry()
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "A test counter",
		})
		counter.Inc()
		registry.MustRegister(counter)

		err := WriteTextfile(path, registry)
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)

		// Ensure permissions are 0644
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test_counter 1")

		// Verify tmp file was removed
		matches, err := filepath.Glob(path + ".*.tmp")
		require.NoError(t, err)
		assert.Empty(t, matches, "Temp files should have been removed")
	})

	t.Run("success_stdout", func(t *testing.T) {
		// Redirect stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		g := &mockGatherer{mfs: mfs}
		err := WriteTextfile("-", g)

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		require.NoError(t, err, "WriteTextfile failed")

		out, _ := io.ReadAll(r)
		expectedContent := "# HELP test_metric test metric help\n# TYPE test_metric gauge\ntest_metric 42\n"

		assert.Equal(t, expectedContent, string(out), "Unexpected stdout content")
	})

	t.Run("error_gather", func(t *testing.T) {
		expectedErr := errors.New("gather failed")
		g := &mockGatherer{err: expectedErr}

		err := WriteTextfile("dummy.prom", g)

		assert.Equal(t, expectedErr, err)
	})

	t.Run("error_mkdir", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a file where a directory would need to be created
		conflictPath := filepath.Join(tempDir, "conflict")
		err := os.WriteFile(conflictPath, []byte("test"), 0644)
		require.NoError(t, err, "Failed to create conflict file")

		path := filepath.Join(conflictPath, "metrics.prom")

		g := &mockGatherer{mfs: mfs}
		err = WriteTextfile(path, g)

		assert.Error(t, err, "Expected error from MkdirAll, got nil")
		if err != nil {
			msg := err.Error()
			isNotDir := strings.Contains(msg, "not a directory") || strings.Contains(msg, "The system cannot find the path specified")
			assert.True(t, isNotDir, "Got unexpected error message: %v", err)
		}
	})

	t.Run("error_create_temp", func(t *testing.T) {
		tempDir := t.TempDir()

		// Make dir read-only so CreateTemp fails
		err := os.Chmod(tempDir, 0555)
		require.NoError(t, err, "Failed to chmod temp dir")

		// Clean up for other tests
		defer os.Chmod(tempDir, 0755)

		path := filepath.Join(tempDir, "metrics.prom")

		g := &mockGatherer{mfs: mfs}
		err = WriteTextfile(path, g)

		assert.Error(t, err, "Expected error from CreateTemp, got nil")
	})
}
