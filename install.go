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

//go:embed plugin.service
var pluginServiceIni []byte

//go:embed plugin.timer
var pluginTimerIni []byte

const (
	installPrefix = "/usr/local"
	serviceName   = "rpi-exporter"
	binDest       = installPrefix + "/bin/" + serviceName
	unitDest      = "/etc/systemd/system/" + serviceName + ".service"

	pluginName        = "rpi-exporter-plugin"
	pluginUnitDest    = "/etc/systemd/system/" + pluginName + ".service"
	pluginTimerDest   = "/etc/systemd/system/" + pluginName + ".timer"
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
	removeBinary(pluginUnitDest)
	slog.Info("uninstalled", "unit", unitDest)
	return nil
}

func InstallPlugin() error {
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

	_ = runSystemctl("stop", pluginName+".timer")
	_ = runSystemctl("stop", pluginName+".service")

	if err := os.MkdirAll(installPrefix+"/bin", 0755); err != nil {
		return err
	}
	if err := os.WriteFile(binDest, data, 0755); err != nil {
		return fmt.Errorf("could not write binary to %s: %w", binDest, err)
	}
	slog.Info("installed binary", "path", binDest)

	if err := os.WriteFile(pluginUnitDest, pluginServiceIni, 0644); err != nil {
		return fmt.Errorf("could not write unit to %s: %w", pluginUnitDest, err)
	}
	slog.Info("wrote unit file", "path", pluginUnitDest)

	if err := os.WriteFile(pluginTimerDest, pluginTimerIni, 0644); err != nil {
		return fmt.Errorf("could not write timer to %s: %w", pluginTimerDest, err)
	}
	slog.Info("wrote timer file", "path", pluginTimerDest)

	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", pluginName + ".timer"},
		{"start", pluginName + ".timer"},
	} {
		if err := runSystemctl(args...); err != nil {
			return err
		}
	}
	return nil
}

func UninstallPlugin() error {
	for _, args := range [][]string{
		{"stop", pluginName + ".timer"},
		{"disable", pluginName + ".timer"},
		{"daemon-reload"},
	} {
		_ = runSystemctl(args...)
	}
	_ = os.Remove(pluginUnitDest)
	_ = os.Remove(pluginTimerDest)
	removeBinary(unitDest)
	slog.Info("uninstalled plugin", "unit", pluginUnitDest, "timer", pluginTimerDest)
	return nil
}

// removeBinary removes the shared binary only when otherUnit is no longer installed.
func removeBinary(otherUnit string) {
	if _, err := os.Stat(otherUnit); os.IsNotExist(err) {
		_ = os.Remove(binDest)
		slog.Info("removed binary", "path", binDest)
	} else {
		slog.Info("keeping binary, other unit still installed", "path", binDest, "unit", otherUnit)
	}
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
