Adhere to the following conventions.

## Development Scripts

Run from project root.

<!-- claudeconfig:begin Project Summary -->
# rpi-exporter

A Prometheus exporter for Raspberry Pi hardware metrics, written in Go. It reads hardware state by shelling out to `vcgencmd` (the Raspberry Pi firmware query tool) and exposes the results as Prometheus metrics.

## What it does

Collects and exposes the following metrics:

| Metric | Source |
|---|---|
| `rpi_voltage_volts{port}` | `vcgencmd measure_volts` |
| `rpi_temperature_celsius` | `vcgencmd measure_temp` |
| `rpi_clock_frequency_hertz{id}` | `vcgencmd measure_clock` |
| `rpi_memory_bytes{id}` | `vcgencmd get_mem` + `/proc/meminfo` (CMA) |
| `rpi_throttled_status` | `vcgencmd get_throttled` (raw bitmask) |
| `rpi_throttled{condition,period}` | same, expanded per-bit |
| `rpi_reset_reason` | `vcgencmd get_rsts` |
| `rpi_gpu_reloc_total{event}` | `vcgencmd mem_reloc_stats` |
| `rpi_gpu_oom_*` | `vcgencmd mem_oom` |

## Main components

### `cmd/rpi-exporter/main.go`
Entry point. Parses flags and dispatches to one of three modes:
- **HTTP server** (default): serves `/metrics` on `:9101`
- **Plugin mode** (`-plugin`): gathers metrics once, writes to a `.prom` file for `node_exporter`'s textfile collector, then exits
- **Install/uninstall** (`-install`, `-install-plugin`, etc.): self-installs as a systemd service or timer

### `collector/` package
- **`commands.go`**: all `vcgencmd` interaction. Each metric has a dedicated `Get*()` function that runs the command and parses its output via regex. A `sync.Mutex` (`systemLock`) serializes all `vcgencmd` calls. `IsRpi()` (checked once via `sync.Once`) gates all command execution.
- **`collector.go`**: implements `prometheus.Collector`. `Describe()` registers all metric descriptors; `Collect()` calls the `Get*()` functions and emits metrics. Per-metric errors are logged and skipped rather than failing the whole collection.

### `textfile.go`
`WriteTextfile(path, gatherer)`: writes Prometheus text-format metrics atomically (write to `.tmp`, then rename). Used for plugin mode.

### `install.go`
Self-installation logic. Reads the current executable and copies it to `/usr/local/bin/rpi-exporter`, then writes embedded systemd unit files and runs `systemctl`. Two install modes: standalone service and textfile plugin timer (runs every 30s).

### `cmd/vcgencmd-stub/`
Fake `vcgencmd` binary for local development. `make test` builds it to `bin/` and prepends that to `PATH` before running tests, so the test suite runs on any machine.

## Key conventions

- **Module path**: `ubunatic.com/rpi-exporter`
- **License**: AGPL-3.0-or-later; all source files carry SPDX headers. REUSE-compliant (`reuse lint`).
- **Go version**: 1.23+
- **No comments on obvious code** — only non-obvious constraints are documented.
- **All scripts run from project root** (per CLAUDE.md).
- **Cross-compilation**: `make build` targets `GOARCH=arm64 GOOS=linux`. Development happens on x86 or a Pi400; tests run locally via the stub, then on-device via `make test-uploads`.
- **Default deploy target**: `RPI_HOST=pi400`, `RPI_USER=uwe` (overridable).

## Before making changes

- `make test` — runs locally using the `vcgencmd` stub; fast, no Pi required.
- `make test-uploads` — compiles, uploads via SCP, and runs tests on the actual Pi.
- Adding a new metric requires: a `Get*()` function in `commands.go`, a stub case in `cmd/vcgencmd-stub/main.go`, a `*Desc` var and `Describe`/`Collect` entries in `collector.go`, and a test case in `commands_test.go`.
- `vcgencmd` calls are serialized — do not call `runVCGenCmd` concurrently.
- Plugin mode and server mode share the same `RPiCollector`; behavior is controlled entirely by flags in `main.go`.
<!-- claudeconfig:end Project Summary -->
