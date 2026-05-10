package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/darthvader/corne-conf/host/config"
	"github.com/godbus/dbus/v5"
)

const (
	bluezService          = "org.bluez"
	objectManagerIface    = "org.freedesktop.DBus.ObjectManager"
	propertiesIface       = "org.freedesktop.DBus.Properties"
	deviceIface           = "org.bluez.Device1"
	gattCharacteristicIfc = "org.bluez.GattCharacteristic1"

	layerCharUUID = "beb5483e-36e1-4688-b7f5-ea07361b26a8"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: $XDG_CONFIG_HOME/corned/config.yaml)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("connect system bus: %v", err)
	}
	defer conn.Close()

	for {
		err = run(ctx, conn, cfg)
		if ctx.Err() != nil {
			return
		}
		log.Printf("retrying after error: %v", err)
		time.Sleep(2 * time.Second)
	}
}

func run(ctx context.Context, conn *dbus.Conn, cfg config.Config) error {
	log.Printf("resolving device target name=%q addr=%q", cfg.DeviceName, cfg.DeviceAddr)

	charPath, initialValue, err := findLayerCharacteristic(conn, cfg)
	if err != nil {
		return err
	}

	log.Printf("initial layer=%q", initialValue)
	if err := writeWaybarJSON(cfg.OutputPath, initialValue); err != nil {
		return err
	}

	obj := conn.Object(bluezService, charPath)
	call := obj.CallWithContext(ctx, gattCharacteristicIfc+".StartNotify", 0)
	if call.Err != nil && !strings.Contains(call.Err.Error(), "InProgress") {
		return fmt.Errorf("start notify: %w", call.Err)
	}
	log.Printf("notifications enabled for %s", charPath)

	matchRule := fmt.Sprintf(
		"type='signal',sender='%s',path='%s',interface='%s',member='PropertiesChanged'",
		bluezService,
		charPath,
		propertiesIface,
	)
	if call := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, matchRule); call.Err != nil {
		return fmt.Errorf("add match: %w", call.Err)
	}
	defer conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, matchRule)

	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	log.Printf("watching %s", charPath)

	for {
		select {
		case <-ctx.Done():
			_ = obj.Call(gattCharacteristicIfc+".StopNotify", 0).Err
			log.Printf("stopping watcher")
			return ctx.Err()
		case sig := <-signals:
			value, ok, err := parseLayerSignal(sig, charPath)
			if err != nil {
				log.Printf("ignoring signal: %v", err)
				continue
			}
			if !ok {
				continue
			}
			log.Printf("layer update=%q", value)
			if err := writeWaybarJSON(cfg.OutputPath, value); err != nil {
				return err
			}
		}
	}
}

func findLayerCharacteristic(conn *dbus.Conn, cfg config.Config) (dbus.ObjectPath, string, error) {
	manager := conn.Object(bluezService, "/")

	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	call := manager.Call(objectManagerIface+".GetManagedObjects", 0)
	if call.Err != nil {
		return "", "", fmt.Errorf("get managed objects: %w", call.Err)
	}
	if err := call.Store(&objects); err != nil {
		return "", "", fmt.Errorf("decode managed objects: %w", err)
	}

	devicePath := findDevicePath(objects, cfg)
	if devicePath == "" {
		return "", "", errors.New("keyboard device not found in BlueZ; pair/connect it first")
	}
	log.Printf("matched device path=%s", devicePath)

	charPath := findCharacteristicPath(objects, devicePath, cfg.Verbose)
	if charPath == "" {
		return "", "", errors.New("layer GATT characteristic not found; ensure the flashed firmware is connected")
	}
	log.Printf("matched layer characteristic path=%s uuid=%s", charPath, layerCharUUID)

	value, err := readCharacteristic(conn, charPath)
	if err != nil {
		return "", "", err
	}

	return charPath, value, nil
}

func findDevicePath(objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant, cfg config.Config) dbus.ObjectPath {
	for path, ifaces := range objects {
		props, ok := ifaces[deviceIface]
		if !ok {
			continue
		}

		addr, _ := variantString(props, "Address")
		name, _ := variantString(props, "Name")
		alias, _ := variantString(props, "Alias")
		connected, _ := variantBool(props, "Connected")
		servicesResolved, _ := variantBool(props, "ServicesResolved")

		debugf(cfg.Verbose, "device candidate path=%s addr=%s name=%q alias=%q connected=%t services_resolved=%t",
			path, addr, name, alias, connected, servicesResolved)

		if cfg.DeviceAddr != "" && !strings.EqualFold(addr, cfg.DeviceAddr) {
			continue
		}
		if cfg.DeviceAddr == "" && name != cfg.DeviceName && alias != cfg.DeviceName {
			continue
		}
		if !connected {
			continue
		}
		return path
	}

	return ""
}

func findCharacteristicPath(objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant, devicePath dbus.ObjectPath, verbose bool) dbus.ObjectPath {
	devicePrefix := string(devicePath) + "/"

	for path, ifaces := range objects {
		if !strings.HasPrefix(string(path), devicePrefix) {
			continue
		}

		props, ok := ifaces[gattCharacteristicIfc]
		if !ok {
			continue
		}

		uuid, _ := variantString(props, "UUID")
		debugf(verbose, "characteristic candidate path=%s uuid=%s", path, uuid)
		if strings.EqualFold(uuid, layerCharUUID) {
			return path
		}
	}

	return ""
}

func readCharacteristic(conn *dbus.Conn, path dbus.ObjectPath) (string, error) {
	obj := conn.Object(bluezService, path)
	var raw []byte
	call := obj.Call(gattCharacteristicIfc+".ReadValue", 0, map[string]dbus.Variant{})
	if call.Err != nil {
		return "", fmt.Errorf("read characteristic: %w", call.Err)
	}
	if err := call.Store(&raw); err != nil {
		return "", fmt.Errorf("decode characteristic value: %w", err)
	}
	value := strings.TrimSpace(string(raw))
	log.Printf("read current layer=%q", value)
	return value, nil
}

func parseLayerSignal(sig *dbus.Signal, expectedPath dbus.ObjectPath) (string, bool, error) {
	if sig == nil || sig.Path != expectedPath || len(sig.Body) != 3 {
		return "", false, nil
	}

	iface, ok := sig.Body[0].(string)
	if !ok || iface != gattCharacteristicIfc {
		return "", false, nil
	}

	changed, ok := sig.Body[1].(map[string]dbus.Variant)
	if !ok {
		return "", false, errors.New("unexpected changed-properties payload")
	}

	valueVariant, ok := changed["Value"]
	if !ok {
		return "", false, nil
	}

	raw, ok := valueVariant.Value().([]byte)
	if !ok {
		return "", false, errors.New("unexpected Value type")
	}

	return strings.TrimSpace(string(raw)), true, nil
}

func writeWaybarJSON(path, layer string) error {
	if layer == "" {
		layer = "UNKNOWN"
	}

	text := layerIcon(layer) + " " + layer
	className := strings.ToLower(strings.ReplaceAll(layer, "/", "-"))
	payload := fmt.Sprintf("{\"text\":%q,\"class\":%q}\n", text, className)
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, []byte(payload), 0o644); err != nil {
		return fmt.Errorf("write temp output: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp output: %w", err)
	}
	log.Printf("wrote %q to %s", layer, path)
	notifyWaybar()
	return nil
}

func layerIcon(layer string) string {
	switch layer {
	case "NORMAL":
		return "⌨️"
	case "NUM/MEDIA", "MEDIA", "MEDIA-UTIL":
		return "🎵"
	case "SYM":
		return "⚡"
	case "MOUSE":
		return "🖱️"
	case "GAMING", "GAME-UTILS", "GAME-MACRO":
		return "🎮"
	case "MODE/BLE":
		return "📶"
	default:
		return ""
	}
}

func notifyWaybar() {
	exec.Command("pkill", "-35", "-x", "waybar").Run()
}

func debugf(enabled bool, format string, args ...any) {
	if enabled {
		log.Printf(format, args...)
	}
}

func variantString(props map[string]dbus.Variant, key string) (string, bool) {
	value, ok := props[key]
	if !ok {
		return "", false
	}
	s, ok := value.Value().(string)
	return s, ok
}

func variantBool(props map[string]dbus.Variant, key string) (bool, bool) {
	value, ok := props[key]
	if !ok {
		return false, false
	}
	b, ok := value.Value().(bool)
	return b, ok
}
