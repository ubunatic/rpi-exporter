package rpiexporter

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"ubunatic.com/rpi-exporter/collector"
)

//go:embed service.ini
var serviceIni []byte

const (
	installPrefix  = "/usr/local"
	serviceName    = "rpi-exporter"
	binDest        = installPrefix + "/bin/" + serviceName
	unitDest       = "/etc/systemd/system/" + serviceName + ".service"
)

func Install() error {
	if !collector.IsRpi() {
		return fmt.Errorf("cannot install on non-RPi system")
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not resolve current binary: %w", err)
	}

	data, err := os.ReadFile(self)
	if err != nil {
		return fmt.Errorf("could not read binary %s: %w", self, err)
	}

	_ = runSystemctl("stop", serviceName+".service") // best-effort before replacing

	if err := os.MkdirAll(installPrefix+"/bin", 0755); err != nil {
		return err
	}
	if err := os.WriteFile(binDest, data, 0755); err != nil {
		return fmt.Errorf("could not write binary to %s: %w", binDest, err)
	}
	slog.Info("installed binary", "path", binDest)

	if err := os.WriteFile(unitDest, serviceIni, 0644); err != nil {
		return fmt.Errorf("could not write unit file to %s: %w", unitDest, err)
	}
	slog.Info("wrote unit file", "path", unitDest)

	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", serviceName + ".service"},
		{"start", serviceName + ".service"},
	} {
		if err := runSystemctl(args...); err != nil {
			return err
		}
	}
	return nil
}

func Uninstall() error {
	for _, args := range [][]string{
		{"stop", serviceName + ".service"},
		{"disable", serviceName + ".service"},
		{"daemon-reload"},
	} {
		_ = runSystemctl(args...) // best-effort
	}
	_ = os.Remove(unitDest)
	_ = os.Remove(binDest)
	slog.Info("uninstalled", "binary", binDest, "unit", unitDest)
	return nil
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	slog.Info("running systemctl", "args", args)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		fmt.Print(string(out))
	}
	return err
}
