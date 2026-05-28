Adhere to the following conventions.

<!-- claudeconfig:begin Project Summary -->
# rpi-exporter

A Prometheus exporter for Raspberry Pi hardware metrics, written in Go 1.23+. Collects voltage, temperature, clock frequencies, memory, throttling status, GPU buffer object stats, and OOM events. Exposes metrics via HTTP or writes them to a node_exporter textfile.

## Data Sources

- `vcgencmd` — primary source for most hardware metrics
- `/proc/meminfo` — CMA memory
- `/sys/kernel/debug/dri/0/bo_stats` — GPU buffer object stats (requires root + debugfs)

## Main Components

| File | Role |
|---|---|
| `collector/commands.go` | All data acquisition. Each `Get*()` calls `runVCGenCmd()` or reads a system file, returns parsed `float64`. `systemLock` (`sync.Mutex`) serializes all `vcgencmd` calls. `IsRpi()` detects vcgencmd on PATH via `sync.Once`. |
| `collector/collector.go` | Implements `prometheus.Collector`. Declares all `*Desc` vars, wires them to `Get*()` in `Describe()` and `Collect()`. Per-metric errors are logged and skipped — never abort collection. |
| `collector/commands_test.go` / `collector/collector_test.go` | Tests run against the stub binary; no mocking. |
| `cmd/rpi-exporter/main.go` | Entry point. Parses flags, dispatches to HTTP server (default), textfile plugin mode (`-plugin`), or install/uninstall. |
| `cmd/vcgencmd-stub/main.go` | Fake `vcgencmd` with hardcoded realistic output for every subcommand. Built to `bin/vcgencmd` by `make build-stub`; prepended to PATH during `make test`. |
| `textfile.go` | `WriteTextfile(path, gatherer)` atomically writes metrics to a `.prom` file via `.tmp` rename; path `"-"` writes to stdout. |
| `install.go` | Copies binary to `/usr/local/bin/rpi-exporter`, installs embedded systemd units via `systemctl`. |

## Key Conventions

**Adding a metric requires four coordinated changes:**
1. Add `Get*()` in `collector/commands.go`
2. Add stub case in `cmd/vcgencmd-stub/main.go`
3. Add `*Desc` var + `Describe()` + `Collect()` entries in `collector/collector.go`
4. Add tests in both test files

**Testing:** always `make test` — builds the stub and injects it into PATH. Never mock. Use `make test-uploads` to compile, SCP to Pi, and run tests on real hardware.

**Metric types:** voltages, temperatures, memory, clocks, GPU BO counts → `GaugeValue`; event counts, OOM totals → `CounterValue`.

**Unit conversions** happen in the collector layer: `GetMemory()` returns bytes (converts from MB); `GetMemOOM()` returns raw values; collector converts to seconds/bytes.

**No global Prometheus registry** — always `prometheus.NewRegistry()`.

**Do not call `runVCGenCmd` concurrently** — serialized via `systemLock`.

**Cross-compilation:** `make build` targets `GOARCH=arm64 GOOS=linux`. Dev workflow: build → SCP → run on Pi.

**License:** AGPL-3.0-or-later; all files carry SPDX headers. Check with `make reuse`.

**Disclaimer:** metric descriptions are based on empirical observation and community knowledge, not official Broadcom/RPi documentation.
<!-- claudeconfig:end Project Summary -->

## Development Scripts

Run from project root.

## Important Notes

**Disclaimer:** metric descriptions and source interpretations are based on empirical
observation and community knowledge, not official Broadcom/RPi documentation.
What a number actually means is often a best guess — verify against your own hardware
if precision matters.
