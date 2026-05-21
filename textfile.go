package rpiexporter

import (
	"io"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const TextfilePath = "/var/lib/prometheus/node-exporter/rpi.prom"

// WriteTextfile gathers metrics and writes them atomically to path.
// Use path "-" to write to stdout.
func WriteTextfile(path string, g prometheus.Gatherer) error {
	mfs, err := g.Gather()
	if err != nil {
		return err
	}
	if path == "-" {
		return encodeText(os.Stdout, mfs)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if err := encodeText(f, mfs); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func encodeText(w io.Writer, mfs []*dto.MetricFamily) error {
	enc := expfmt.NewEncoder(w, expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			return err
		}
	}
	return nil
}
