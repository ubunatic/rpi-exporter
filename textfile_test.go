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

	dto "github.com/prometheus/client_model/go"
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
	// Create some dummy metrics
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

	t.Run("success_file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "metrics.prom")

		g := &mockGatherer{mfs: mfs}
		err := WriteTextfile(path, g)
		if err != nil {
			t.Fatalf("WriteTextfile failed: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		expectedContent := "# HELP test_metric test metric help\n# TYPE test_metric gauge\ntest_metric 42\n"
		if string(content) != expectedContent {
			t.Errorf("Unexpected output file content.\nGot: %q\nWant: %q", string(content), expectedContent)
		}

		// Verify tmp file was removed
		tmpPath := path + ".tmp"
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			t.Errorf("Temp file %s still exists", tmpPath)
		}
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

		if err != nil {
			t.Fatalf("WriteTextfile failed: %v", err)
		}

		out, _ := io.ReadAll(r)
		expectedContent := "# HELP test_metric test metric help\n# TYPE test_metric gauge\ntest_metric 42\n"

		if string(out) != expectedContent {
			t.Errorf("Unexpected stdout content.\nGot: %q\nWant: %q", string(out), expectedContent)
		}
	})

	t.Run("error_gather", func(t *testing.T) {
		expectedErr := errors.New("gather failed")
		g := &mockGatherer{err: expectedErr}

		err := WriteTextfile("dummy.prom", g)

		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("error_mkdir", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a file where a directory would need to be created
		conflictPath := filepath.Join(tempDir, "conflict")
		if err := os.WriteFile(conflictPath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create conflict file: %v", err)
		}

		path := filepath.Join(conflictPath, "metrics.prom")

		g := &mockGatherer{mfs: mfs}
		err := WriteTextfile(path, g)

		if err == nil {
			t.Error("Expected error from MkdirAll, got nil")
		} else if !strings.Contains(err.Error(), "not a directory") && !strings.Contains(err.Error(), "The system cannot find the path specified") {
			// Allow for different OS error messages
			t.Logf("Got error: %v", err)
		}
	})

	t.Run("error_open_file", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "metrics.prom")
		tmpPath := path + ".tmp"

		// Create a directory where the temp file would be written, causing an error
		if err := os.Mkdir(tmpPath, 0755); err != nil {
			t.Fatalf("Failed to create conflict dir: %v", err)
		}

		g := &mockGatherer{mfs: mfs}
		err := WriteTextfile(path, g)

		if err == nil {
			t.Error("Expected error from OpenFile, got nil")
		}
	})
}
