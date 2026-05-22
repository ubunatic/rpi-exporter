// SPDX-FileCopyrightText: 2026 Uwe Jugel <uwe@ubunatic.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	rpiexporter "ubunatic.com/rpi-exporter"
	"ubunatic.com/rpi-exporter/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("rpi-exporter", flag.ContinueOnError)
	port := fs.String("port", ":9101", "Port to listen on")
	rpi := fs.Bool("rpi", false, "Check if running on a Raspberry Pi and exit")
	install := fs.Bool("install", false, "Install rpi-exporter as a systemd service")
	uninstall := fs.Bool("uninstall", false, "Uninstall rpi-exporter systemd service")
	plugin := fs.Bool("plugin", false, "Write metrics to textfile collector and exit")
	textfile := fs.String("textfile", rpiexporter.TextfilePath, "Textfile collector output path (used with -plugin; use - for stdout)")
	installPlugin := fs.Bool("install-plugin", false, "Install rpi-exporter as a systemd timer (textfile plugin)")
	uninstallPlugin := fs.Bool("uninstall-plugin", false, "Uninstall rpi-exporter systemd timer plugin")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *rpi {
		if collector.IsRpi() {
			slog.Info("Running on a Raspberry Pi")
		} else {
			slog.Error("Not running on a Raspberry Pi")
		}
		return 0
	}

	if *install {
		if err := rpiexporter.Install(); err != nil {
			slog.Error("Failed to install", "error", err)
			return 1
		}
		return 0
	}

	if *uninstall {
		if err := rpiexporter.Uninstall(); err != nil {
			slog.Error("Failed to uninstall", "error", err)
			return 1
		}
		return 0
	}

	if *installPlugin {
		if err := rpiexporter.InstallPlugin(); err != nil {
			slog.Error("Failed to install plugin", "error", err)
			return 1
		}
		return 0
	}

	if *uninstallPlugin {
		if err := rpiexporter.UninstallPlugin(); err != nil {
			slog.Error("Failed to uninstall plugin", "error", err)
			return 1
		}
		return 0
	}

	if *plugin {
		reg := prometheus.NewRegistry()
		reg.MustRegister(collector.NewRPiCollector())
		if err := rpiexporter.WriteTextfile(*textfile, reg); err != nil {
			slog.Error("Failed to write textfile", "path", *textfile, "error", err)
			return 1
		}
		return 0
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewRPiCollector())

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	slog.Info("Starting rpi-exporter", "port", *port, "version", version)
	if err := http.ListenAndServe(*port, mux); err != nil {
		slog.Error("Failed to start server", "error", err)
		return 1
	}
	return 0
}
