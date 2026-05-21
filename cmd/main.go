package main

import (
	"flag"
	"log/slog"
	"net/http"

	rpiexporter "ubunatic.com/rpi-exporter"
	"ubunatic.com/rpi-exporter/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	port := flag.String("port", ":9101", "Port to listen on")
	rpi := flag.Bool("rpi", false, "Check if running on a Raspberry Pi and exit")
	install := flag.Bool("install", false, "Install rpi-exporter as a systemd service")
	uninstall := flag.Bool("uninstall", false, "Uninstall rpi-exporter systemd service")
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

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.NewRPiCollector())

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	slog.Info("Starting rpi-exporter", "port", *port)
	if err := http.ListenAndServe(*port, nil); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}
