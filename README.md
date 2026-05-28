# rpi-exporter

A Prometheus exporter for Raspberry Pi hardware metrics via `vcgencmd`.

## Metrics

| Metric | Description |
|--------|-------------|
| `rpi_voltage_volts{port}` | Voltage of core, sdram_c, sdram_p, sdram_i |
| `rpi_temperature_celsius` | SoC temperature |
| `rpi_clock_frequency_hertz{id}` | Clock frequency for arm, core, v3d, isp, h264, pixel, uart |
| `rpi_memory_bytes{id}` | Memory values — see table below |
| `rpi_throttled_status` | Raw throttled bitmask from `vcgencmd get_throttled` |
| `rpi_throttled{condition,period}` | Per-bit throttle flags (under_voltage, freq_capped, throttled, soft_temp_limit × now/since_boot) |
| `rpi_reset_reason` | Reset reason bitmask from `vcgencmd get_rsts` |
| `rpi_gpu_reloc_total{event}` | VideoCore relocatable heap event counts since boot (alloc_failures, compactions, legacy_block_fails) |
| `rpi_gpu_oom_events_total` | VideoCore OOM event count since boot |
| `rpi_gpu_oom_lifetime_bytes` | Total bytes required by VideoCore OOM handler since boot |
| `rpi_gpu_oom_handler_seconds_total` | Total time in VideoCore OOM handler since boot |
| `rpi_gpu_oom_handler_max_seconds` | Max time spent in a single VideoCore OOM handler call |
| `rpi_gpu_bo_objects` | Number of V3D GPU buffer objects currently allocated |
| `rpi_gpu_bo_bytes` | Total bytes in V3D GPU buffer objects |

> **Disclaimer:** metric descriptions and source interpretations are based on empirical
> observation and community knowledge, not official Broadcom/RPi documentation. The
> "what does this number actually mean" is often a best guess — the CMA story above is
> a good example where the obvious reading (`vcgencmd get_mem cma`) turned out to be
> the wrong source for a common configuration. Treat descriptions as working hypotheses
> and verify against your own hardware if precision matters.

### `rpi_memory_bytes{id}` label values

| id | Source | Description |
|----|--------|-------------|
| `arm` | `vcgencmd get_mem arm` | RAM assigned to the ARM cores |
| `gpu` | `vcgencmd get_mem gpu` | RAM assigned to the VideoCore GPU firmware |
| `malloc` | `vcgencmd get_mem malloc` | VideoCore heap currently in use |
| `malloc_total` | `vcgencmd get_mem malloc_total` | VideoCore heap total size |
| `reloc` | `vcgencmd get_mem reloc` | VideoCore relocatable heap in use |
| `reloc_total` | `vcgencmd get_mem reloc_total` | VideoCore relocatable heap total |
| `cma` | `vcgencmd get_mem cma` | VideoCore-managed CMA in use (0 when kernel CMA is used instead) |
| `cma_total` | `vcgencmd get_mem cma_total` | VideoCore-managed CMA total (0 when kernel CMA is used instead) |
| `cma_reserved` | `/proc/meminfo CmaTotal` | Kernel CMA reserved (e.g. via `dtoverlay=vc4-kms-v3d,cma-512`) |
| `cma_free` | `/proc/meminfo CmaFree` | Kernel CMA currently free |

**Note on CMA:** `dtoverlay=vc4-kms-v3d,cma-N` reserves kernel CMA, not VideoCore-managed CMA.
In that case `vcgencmd get_mem cma/cma_total` returns 0. The actual reservation is in `/proc/meminfo`
as `CmaTotal`/`CmaFree`, exposed here as `cma_reserved`/`cma_free`.

**Note on `reloc` vs `rpi_gpu_bo_bytes`:** Both measure the same pool from different angles —
`reloc` is what the VideoCore firmware reports via `vcgencmd`, `rpi_gpu_bo_bytes` is what the
V3D DRM driver reports via `/sys/kernel/debug/dri/0/bo_stats`. They should be approximately equal.
`malloc` is the VideoCore firmware's own private heap on top of that.

## Usage

### Standalone HTTP server

```
rpi-exporter [-port :9101]
```

Serves metrics at `http://<host>:9101/metrics`.

### Textfile plugin for node_exporter

```
rpi-exporter -plugin [-textfile /var/lib/prometheus/node-exporter/rpi.prom]
```

Writes metrics to the node_exporter textfile directory and exits. Use `-textfile -` to print to stdout.

## Installation

Requires a Raspberry Pi with `vcgencmd` available.

### Via `go install` (on the Pi)

```sh
go install ubunatic.com/rpi-exporter/cmd/rpi-exporter@latest

# sudo is required since ~/go/bin is not in sudo's PATH
# use the full path or a relative path from your home directory
sudo ~/go/bin/rpi-exporter -install-plugin   # systemd timer (node_exporter plugin)
sudo ~/go/bin/rpi-exporter -install          # standalone systemd service
```

### Via Makefile (cross-compile from a dev machine)

Cross-compiles for arm64, uploads to the Pi, and installs.

```sh
make install           # build, upload, install as rpi-exporter.service
make uninstall         # stop and remove
```

### Systemd timer (node_exporter plugin)

Runs every 30 seconds and writes metrics to the node_exporter textfile directory.

```sh
make install-plugin    # build, upload, install as rpi-exporter-plugin.timer
make uninstall-plugin  # stop and remove
```

## Development

```sh
make test              # run tests locally using a vcgencmd stub
make build             # cross-compile for arm64/linux
make upload            # build and upload binaries to the Pi (RPI_HOST=pi400 RPI_USER=uwe)
make test-uploads      # build, upload, and run tests on the Pi
make query             # curl the metrics endpoint on the Pi (port 9101)
make query-node        # grep rpi_ metrics from node_exporter on the Pi (port 9100)
make query-plugin      # grep rpi_ metrics from textfile output on the Pi
```

Override the target host: `make upload RPI_HOST=mypi RPI_USER=pi`

### Adding a new metric

1. Add a `Get*()` function in `collector/commands.go`
2. Add a stub case in `cmd/vcgencmd-stub/main.go` (for local testing)
3. Add a `*Desc` var, `Describe()` entry, and `Collect()` entry in `collector/collector.go`
4. Add a test in `collector/commands_test.go` and `collector/collector_test.go`

## Requirements

- Go 1.23+
- Raspberry Pi with `vcgencmd` (Raspberry Pi OS)
- For plugin mode: `node_exporter` with `--collector.textfile.directory=/var/lib/prometheus/node-exporter`
- For `rpi_gpu_bo_*`: debugfs must be mounted (`/sys/kernel/debug`) and the process must run as root

## License

AGPL-3.0-or-later — see [LICENSES/AGPL-3.0-or-later.txt](LICENSES/AGPL-3.0-or-later.txt)
