// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package rpiexporter

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeText_Success(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_metric",
		Help: "A test metric",
	})
	gauge.Set(42)

	reg := prometheus.NewRegistry()
	reg.MustRegister(gauge)

	mfs, err := reg.Gather()
	require.NoError(t, err)

	var buf bytes.Buffer
	err = encodeText(&buf, mfs)
	require.NoError(t, err)

	output := buf.String()
	require.True(t, strings.Contains(output, "# HELP test_metric A test metric"))
	require.True(t, strings.Contains(output, "# TYPE test_metric gauge"))
	require.True(t, strings.Contains(output, "test_metric 42"))
}

func TestEncodeText_Empty(t *testing.T) {
	var buf bytes.Buffer
	err := encodeText(&buf, []*dto.MetricFamily{})
	require.NoError(t, err)
	require.Empty(t, buf.String())
}

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}

func TestEncodeText_WriterError(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_metric",
		Help: "A test metric",
	})
	gauge.Set(42)

	reg := prometheus.NewRegistry()
	reg.MustRegister(gauge)

	mfs, err := reg.Gather()
	require.NoError(t, err)

	ew := &errorWriter{}
	err = encodeText(ew, mfs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "write error")
}

func TestWriteTextfile(t *testing.T) {
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
}
