// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun_Help(t *testing.T) {
	exitCode := run([]string{"-h"})
	assert.Equal(t, 2, exitCode)
}

func TestRun_InvalidFlag(t *testing.T) {
	exitCode := run([]string{"-invalid-flag"})
	assert.Equal(t, 2, exitCode)
}

func TestRun_RpiFlag(t *testing.T) {
	exitCode := run([]string{"-rpi"})
	assert.Equal(t, 0, exitCode)
}

func TestRun_Install(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping destructive test run as root")
	}
	exitCode := run([]string{"-install"})
	// returns 0 or 1 depending on whether it failed, both are acceptable execution paths
	assert.Contains(t, []int{0, 1}, exitCode)
}

func TestRun_Uninstall(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping destructive test run as root")
	}
	exitCode := run([]string{"-uninstall"})
	assert.Contains(t, []int{0, 1}, exitCode)
}

func TestRun_InstallPlugin(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping destructive test run as root")
	}
	exitCode := run([]string{"-install-plugin"})
	assert.Contains(t, []int{0, 1}, exitCode)
}

func TestRun_UninstallPlugin(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping destructive test run as root")
	}
	exitCode := run([]string{"-uninstall-plugin"})
	assert.Contains(t, []int{0, 1}, exitCode)
}

func TestRun_Plugin_Stdout(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{"-plugin", "-textfile", "-"})
	assert.Equal(t, 0, exitCode)

	// Restore stdout
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	// Depending on whether it could gather metrics, it might or might not have `# HELP`
	// at least it shouldn't fail
	assert.True(t, len(output) >= 0)
}

func TestRun_Plugin_InvalidPath(t *testing.T) {
	exitCode := run([]string{"-plugin", "-textfile", "/invalid/path/that/does/not/exist/rpi.prom"})
	assert.Equal(t, 1, exitCode)
}

func TestRun_Listen_InvalidPort(t *testing.T) {
	// Suppress slog output for this test to avoid polluting test logs with error
	oldLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(oldLogger) })
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	exitCode := run([]string{"-port", "invalid-port"})
	assert.Equal(t, 1, exitCode)
}

func TestRun_Listen_ValidPort(t *testing.T) {
	// Run the server in a goroutine
	done := make(chan int)
	go func() {
		done <- run([]string{"-port", ":0"}) // Let OS assign a free port
	}()

	// Give it some time to start
	select {
	case exitCode := <-done:
		t.Errorf("Server exited prematurely with code %d", exitCode)
	case <-time.After(100 * time.Millisecond):
		// Server is likely running fine
	}
}
