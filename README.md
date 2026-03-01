# Retaliq Domain Helper Setup Script

This repository contains a helper script (`setup.sh`) used to manage the
`retaliq-domain` binary and its accompanying systemd service on Unix-like
systems. The script is intended to be run as `root` (via `sudo`) and
uses basic Unix tools for configuration file manipulation (no JSON parser required).

## Prerequisites

- A Unix-like system with `systemd`.
- `bash` (the script uses bashisms).
- `openssl` (used for API key generation).
- The `retaliq-domain` binary placed alongside `setup.sh` or adjust `BIN` env.

## Installation

Place the script and binary in the same directory, then run:

```sh
sudo ./setup.sh install
```

This will create a default configuration file at `/etc/retaliq-domain/config.conf`,
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

### CLI commands

Because the helper usually runs as a long‑lived service, you can manipulate
its configuration directly via the executable instead of talking to the
HTTP API. The following invocations are available:

```sh
# add a single address to allowed_ips
retaliq-domain add-ip 1.2.3.4

# generate (or renew) API key and print it
retaliq-domain gen-key

# dump current config values to stdout
retaliq-domain show
```

Under the hood these commands load the same config file used by the service
and update it atomically; nothing is sent over the network.

## Configuration

The default configuration file (`/etc/retaliq-domain/config.conf`) is a
simple key/value list with two settings. Blank lines and lines starting with
`#` are ignored.  If an existing file lacks an `api_key`, the helper will
automatically generate one and update the file when it starts.

```ini
# API key used to protect the hosts helper; must be non‑empty
api_key=<random generated key or empty>

# comma-separated client addresses allowed to call the service
allowed_ips=127.0.0.1
```

**Important:** the `api_key` **must be non‑empty** or the helper will exit on
startup. The package’s postinst initially writes an empty value, so you
should either generate one yourself or run the setup script to regenerate
the key:

```sh
# create a key and restart service
sudo ./setup.sh regen-key
sudo systemctl restart retaliq-domain.service
```

Alternatively edit `/etc/retaliq-domain/config.conf` by hand and give `api_key` a
value, then reload the service.

`set-allowed-ip-hosts` will update the `allowed_ips` array while preserving
internal known addresses (loopback, host IP, docker bridge).

## Notes

- The script enforces running as root and will exit otherwise.
- An absolute path to the binary is automatically computed but can be
  overridden via the `BIN` environment variable.
* `openssl` is required and typically already installed on Unix-like
  systems.

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
-config    path to config file (default /etc/retaliq-domain/config.conf)
-apikey    API key (overrides config file)
-allowed   comma-separated allowed IP addresses
-save-config  write effective config back to file and exit
```

The file format is key/value as shown above; command-line flags override
file values.

When invoked with `-save-config`, the helper writes out whatever key/ip list
is currently in effect and then exits, useful for bootstrapping.

### Example invocation

```sh
./retaliq-domain -config /etc/retaliq-domain/config.conf
```

or to generate a config:

```sh
./retaliq-domain -apikey mysecret -allowed 127.0.0.1,10.0.0.1 -save-config
```

## Packaging as a Debian package

The project includes a `debian/` directory that describes how to build a
`.deb` package suitable for installation via `apt`.

A helper script at `apps/domain/debian/build.sh` automates versioning and invokes dpkg-buildpackage. The `debian/` directory also contains the systemd unit (`retaliq-domain.service`).
It derives the package version from the latest Git tag (stripping a leading
`v` if present), updates `debian/changelog`, and then builds.  Simply run:

```sh
cd apps/domain
./debian/build.sh
```

Dependencies: `git`, `debhelper` (and `devscripts` for `dch` if you
want automatic changelog updates), `dpkg-dev`, `golang-go` (for building the
binary), and `build-essential`.

Examples:

```sh
./debian/build.sh            # build using current tag or 0.1.0
# resulting .deb and other artefacts appear in the dist/ subdirectory
```

The `dist/` directory is the canonical output location; when you publish
packages via an APT repository point the repository at `dist/` and
regenerate the `Packages.gz` index there. For instance:

```sh
cd apps/domain/dist
apt-ftparchive packages . > Packages
gzip -c Packages > Packages.gz
# serve this folder over HTTP and let clients add its URL
```

This will compile the Go binary and produce
`../retaliq-domain_<version>_<arch>.deb`.  Install with:

```sh
sudo dpkg -i ../retaliq-domain_*.deb
```

Once installed, the service is automatically enabled and started.  The
package’s maintenance scripts handle configuration file creation and
removal.

To make the package available via `apt-get`, host the `.deb` in an APT
repository (see Debian's packaging guide) and instruct users to add its
`deb` line to `/etc/apt/sources.list`.

Additional repository management (signing, Release files) is outside the
scope of this repo but can be automated in CI.



## Example Usage

You can manage the service either via the helper script or directly with
`systemctl` once the package is installed.

```sh
# using the setup helper
sudo ./setup.sh status          # show status + config
sudo ./setup.sh start           # install (if necessary) and launch
sudo ./setup.sh stop            # stop but keep unit installed
sudo ./setup.sh reload          # restart service
sudo ./setup.sh regenerate-key  # new API key
sudo ./setup.sh set-allowed-ip-hosts 10.0.0.5,8.8.8.8
sudo ./setup.sh uninstall       # full cleanup

# or use systemctl directly
sudo systemctl status retaliq-domain.service   # check current state
sudo systemctl start retaliq-domain.service    # start service
sudo systemctl stop retaliq-domain.service     # stop it
sudo systemctl daemon-reload                   # reload unit after changes
```

Feel free to modify the script or README to suit your environment.
