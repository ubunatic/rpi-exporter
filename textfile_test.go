package rpiexporter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
