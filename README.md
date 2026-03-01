# Retaliq Domain Helper Setup Script

This repository contains a helper script (`setup.sh`) used to manage the
`retaliq-domain` binary and its accompanying systemd service on Unix-like
systems. The script is intended to be run as `root` (via `sudo`) and
depends on `jq` for configuration file manipulation.

## Prerequisites

- A Unix-like system with `systemd`.
- `bash` (the script uses bashisms).
- `jq` (the script will attempt to install it using `apt-get` if missing).
- The `retaliq-domain` binary placed alongside `setup.sh` or adjust `BIN` env.

## Installation

Place the script and binary in the same directory, then run:

```sh
sudo ./setup.sh install
```

This will create a default configuration file at `/etc/retaliq-domain.json`,
enable (but not start) the `retaliq-domain.service` unit.

## Available Commands

The script exposes the following commands:

| Command                  | Description |
|--------------------------|-------------|
| `install`               | Alias for `start` (install and start the service). |
| `start`                 | Install if missing, then start the service and
|                         | show `api_key`, `allowed_ips` and port. |
| `stop`                  | Stop the service, leaving it installed. |
| `reload`                | Stop, then start (ensures service installed). |
| `uninstall`             | Completely remove the service, config, and any running
|                         | processes. |
| `status`                | Display `systemctl status` for the unit and print the
|                         | current config values. |
| `regenerate-key`        | Generate a new API key in the config and reload service. |
| `set-allowed-ip-hosts`  | Prompt or accept comma-separated IPs to update allowed
|                         | host list (appends known local addresses). |
| `help`                  | Show usage message. |

## Configuration

The default configuration file (`/etc/retaliq-domain.json`) contains two
fields:

```json
{
  "api_key": "<random generated key>",
  "allowed_ips": ["127.0.0.1"]
}
```

`set-allowed-ip-hosts` will update the `allowed_ips` array while preserving
internal known addresses (loopback, host IP, docker bridge).

## Notes

- The script enforces running as root and will exit otherwise.
- An absolute path to the binary is automatically computed but can be
  overridden via the `BIN` environment variable.
- `jq` is installed automatically on Debian/Ubuntu systems via `apt-get` if
  not present; otherwise installation must be manual.

## Program overview

The `retaliq-domain` binary is the core helper that the setup script
manages. It provides an HTTP API for manipulating the system's `/etc/hosts`
file, mainly so containers and services can request DNS entries at runtime.

### Operation

* **Port**: defaults to `8888` but may be changed via the `-port`
  command-line flag.
* **Hosts file**: writes to `/etc/hosts` on Unix; a Windows path is encoded in
  `defaultHostsPath()`.
* **Logging**: minimal info to stdout/stderr; systemd captures logs when
  running as a service.

### Endpoints

* `POST /hosts` – expects a JSON array of hostnames; the helper adds each
  name to `/etc/hosts` if not already present. Requires header
  `X-Api-Key: <key>`.
* `GET /version` – returns program version and build info (if implemented).
* Additional endpoints may exist (check source in `handler.go`).

### Configuration and CLI flags

Flags (see `main.go`):

```
-config    path to JSON config file (default /etc/retaliq-domain.json)
-apikey    API key (overrides config file)
-allowed   comma-separated allowed IP addresses
-save-config  write effective config back to file and exit
```

A configuration file is a simple JSON object containing `api_key` and
`allowed_ips` (array). Command-line flags override file values.

When invoked with `-save-config`, the helper writes out whatever key/ip list
is currently in effect and then exits, useful for bootstrapping.

### Example invocation

```sh
./retaliq-domain -config /etc/retaliq-domain.json
```

or to generate a config:

```sh
./retaliq-domain -apikey mysecret -allowed 127.0.0.1,10.0.0.1 -save-config
```



## Example Usage

```sh
sudo ./setup.sh status
sudo ./setup.sh start             # install & run
sudo ./setup.sh stop              # stop but keep installed
sudo ./setup.sh reload            # restart service
sudo ./setup.sh regenerate-key    # new API key
sudo ./setup.sh set-allowed-ip-hosts 10.0.0.5,8.8.8.8
sudo ./setup.sh uninstall         # full cleanup
```

Feel free to modify the script or README to suit your environment.
