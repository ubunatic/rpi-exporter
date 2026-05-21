# rpi-exporter

A Prometheus exporter for Raspberry Pi hardware metrics via `vcgencmd`.

## Metrics

| Metric | Description |
|--------|-------------|
| `rpi_voltage_volts{port}` | Voltage of core, sdram_c, sdram_p, sdram_i |
| `rpi_temperature_celsius` | SoC temperature |
| `rpi_clock_frequency_hertz{id}` | Clock frequency for arm, core, v3d, isp, h264, pixel, uart |
| `rpi_memory_bytes{id}` | Memory split between arm and gpu |
| `rpi_throttled_status` | Raw throttled bitmask from `vcgencmd get_throttled` |
| `rpi_throttled{condition,period}` | Per-bit throttle flags (under_voltage, freq_capped, throttled, soft_temp_limit × now/since_boot) |
| `rpi_reset_reason` | Reset reason bitmask from `vcgencmd get_rsts` |

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

## Requirements

- Go 1.23+
- Raspberry Pi with `vcgencmd` (Raspberry Pi OS)
- For plugin mode: `node_exporter` with `--collector.textfile.directory=/var/lib/prometheus/node-exporter`

## License

AGPL-3.0-or-later — see [LICENSES/AGPL-3.0-or-later.txt](LICENSES/AGPL-3.0-or-later.txt)
