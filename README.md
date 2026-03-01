# retaliq-domain

`retaliq-domain` is a small HTTP helper service used by the
Retaliq infrastructure to keep a machine's hosts file in sync with a
centralised list of domains.  A controller (typically running in the
cloud) POSTs a JSON array of hostnames to `/hosts` and the service
replaces an inline block in `/etc/hosts` (or the platform equivalent)
with the supplied entries.

The binary can also be used as a simple configuration helper from the
command line and is distributed as a Debian package under the name
`retaliq-domain`.

---

## Features

* REST endpoint at `/hosts` accepting POST requests with `X-Api-Key` header
* IP whitelisting by origin address
* Minimal file-based configuration (`key=value`)
* Atomic updates of the hosts file preserving existing permissions
* CLI helpers for configuration and systemd control
* Cross‑platform support (Linux, macOS, Windows)

---

## Building

The code is written in Go and requires Go 1.18+ on your `PATH`.

```sh
# run the unit tests
make test

# build for the current platform
go build -o retaliq-domain .

# or use the convenience targets
make all       # build everything (deb, windows, macos)
make deb       # produce a Debian package
make windows   # cross‑compile Windows executable
make macos     # cross‑compile macOS executable
```

A Debian package will be created under `dist/` and installs a
`retaliq-domain.service` systemd unit (see `debian/` directory).

---

## Configuration

`retaliq-domain` reads a simple `key=value` file.  Comments begin with
`#` and are ignored.  Example:

```
# /etc/retaliq-domain/config.conf
api_key = s3cr3t
allowed_ips = 127.0.0.1,10.0.0.5
```

* `api_key` – the token clients must supply in the `X-Api-Key` header.
  If the key is missing on first run the service generates one and
  persists it back to the file.
* `allowed_ips` – comma‑separated list of addresses permitted to POST.

By default the configuration file is located at
`/etc/retaliq-domain/config.conf` (on Windows the equivalent under
`%ProgramData%`).  You can override this with `-config` when starting
or by passing the `-save-config` flag to write effective settings.

### Command‑line helpers

The same binary acts as a small admin tool.  When invoked with a
positional command it performs the action and exits:

```sh
# add a new source address
sudo retaliq-domain add-ip 192.168.1.100

# generate / rotate the API key
sudo retaliq-domain gen-key
# (alias: generate-key)

# show the current configuration
sudo retaliq-domain show

# control the systemd unit (linux only)
sudo retaliq-domain status
sudo retaliq-domain start
sudo retaliq-domain stop
```

The `-apikey`, `-allowed` and `-port` flags may be used to temporarily
override the settings read from the file.

---

## HTTP API

Only `POST /hosts` is implemented.

* Request body: JSON array of hostnames, e.g. `[
  "foo.local",
  "bar.internal"
]`
* Header `X-Api-Key` must match the configured key.
* The client must connect from an address listed in `allowed_ips`.

On success the service updates the hosts file, replacing the existing
inline block marked by
`# BEGIN RETALIQHOSTS inline` … `# END RETALIQHOSTS inline` and
returns `200 OK`.  Errors are indicated with the usual HTTP status
codes (401, 403, 400, 500).


---

## Hosts file format

Only the contents between the two marker comments are managed.  Entries
are written with both IPv4 and IPv6 loopback addresses:

```
# BEGIN RETALIQHOSTS inline
127.0.0.1  example.local
::1        example.local
# END RETALIQHOSTS inline
```

Any other lines in the file are preserved verbatim.

---

## Testing

Unit tests live beside the code.  Run them with:

```sh
go test ./...
```

There is no external dependency; the tests exercise configuration
parsing, CLI handling and the HTTP handler logic.

---

## Packaging and distribution

The `debian/` directory contains a basic Debian packaging skeleton.  The
`Makefile` target `make deb` invokes `debian/build.sh` which produces a
`.deb` suitable for installation on Debian/Ubuntu systems.  The package
installs the binary and systemd unit described above.

For Windows/macOS the `make windows` and `make macos` targets create
simple stand‑alone executables.

---

## License

This project is released under the MIT License.  See the top‑level
`LICENSE` file for details.
