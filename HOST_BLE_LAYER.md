# Corne BLE Layer Watcher

Exposes the active keyboard layer over a custom BLE GATT characteristic.

- Service UUID: `4fafc201-1fb5-459e-8fcc-c5c9c331914b`
- Characteristic UUID: `beb5483e-36e1-4688-b7f5-ea07361b26a8`

## Components

- **cornd** — daemon, connects to BlueZ, subscribes to layer notifications, writes Waybar JSON
- **cornectl** — CLI client for querying state

## Config

Config is read from `~/.config/corned/config.yaml` (XDG-compatible).

```yaml
# ~/.config/corned/config.yaml
device_name: "Corne Keyboard"   # BlueZ device name to match
device_addr: ""                 # optional, pin by MAC address
output_path: "~/.cache/corne-layer.json"  # Waybar JSON output
verbose: false                  # enable debug logging
```

If the file doesn't exist, defaults are used. No config file is required.

## Build

```bash
# daemon
cd host/cornd
go mod tidy
go build -o ~/.local/bin/cornd .

# CLI
cd host/cornectl
go mod tidy
go build -o ~/.local/bin/cornectl .
```

## Systemd user service

```bash
mkdir -p ~/.config/systemd/user
cp systemd/cornd.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now cornd.service
```

## Runit

```bash
mkdir -p ~/.config/runit/sv
cp -r runit/cornd ~/.config/runit/sv/
chmod +x ~/.config/runit/sv/cornd/run
ln -s ~/.config/runit/sv/cornd ~/.config/runit/runsvdir/default/
```

## Waybar

Merge `waybar/corne-layer.jsonc` into your Waybar config and add `custom/corne-layer` to the modules list.

## cornectl

```bash
# show current layer
cornectl get layer

# show effective config
cornectl get config
```
