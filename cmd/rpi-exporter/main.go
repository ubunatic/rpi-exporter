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
	port            := flag.String("port", ":9101", "Port to listen on")
	rpi             := flag.Bool("rpi", false, "Check if running on a Raspberry Pi and exit")
	install         := flag.Bool("install", false, "Install rpi-exporter as a systemd service")
	uninstall       := flag.Bool("uninstall", false, "Uninstall rpi-exporter systemd service")
	plugin          := flag.Bool("plugin", false, "Write metrics to textfile collector and exit")
	textfile        := flag.String("textfile", rpiexporter.TextfilePath, "Textfile collector output path (used with -plugin; use - for stdout)")
	installPlugin   := flag.Bool("install-plugin", false, "Install rpi-exporter as a systemd timer (textfile plugin)")
	uninstallPlugin := flag.Bool("uninstall-plugin", false, "Uninstall rpi-exporter systemd timer plugin")
	flag.Parse()

	if *rpi {
		if collector.IsRpi() {
			slog.Info("Running on a Raspberry Pi")
		} else {
			slog.Error("Not running on a Raspberry Pi")
		}
		return
	}

	if *install {
		if err := rpiexporter.Install(); err != nil {
			slog.Error("Failed to install", "error", err)
		}
		return
	}

	if *uninstall {
		if err := rpiexporter.Uninstall(); err != nil {
			slog.Error("Failed to uninstall", "error", err)
		}
		return
	}

	if *installPlugin {
		if err := rpiexporter.InstallPlugin(); err != nil {
			slog.Error("Failed to install plugin", "error", err)
		}
		return
	}

	if *uninstallPlugin {
		if err := rpiexporter.UninstallPlugin(); err != nil {
			slog.Error("Failed to uninstall plugin", "error", err)
		}
		return
	}

	if *plugin {
		reg := prometheus.NewRegistry()
		reg.MustRegister(collector.NewRPiCollector())
		if err := rpiexporter.WriteTextfile(*textfile, reg); err != nil {
			slog.Error("Failed to write textfile", "path", *textfile, "error", err)
			os.Exit(1)
		}
		return
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewRPiCollector())

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	slog.Info("Starting rpi-exporter", "port", *port, "version", version)
	if err := http.ListenAndServe(*port, nil); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}
