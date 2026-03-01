#!/usr/bin/env bash
# simple helper for the domain helper service on Unix-like systems.
# Intended to mimic a subset of the Windows setup.ps1 behaviour: create a
# persistent configuration file, register/reload a systemd service, and
# regenerate the API key.

set -euo pipefail

# ensure running as root/sudo
if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root (sudo)." >&2
    exit 1
fi

CONFIG=${CONFIG:-/etc/retaliq-domain/config.conf}
# ensure BIN is absolute path to the helper binary, even when script is run via relative path
BIN=${BIN:-$(dirname "$(readlink -f "$0")")/dist/retaliq-domain}
SERVICE_NAME=retaliq-domain.service

check_openssl() {
    if ! command -v openssl >/dev/null 2>&1; then
        echo "openssl is required and not installed. please install it and rerun."
        exit 1
    fi
}

usage() {
    cat <<EOF
Usage: $0 <command>
Commands:
  install              install the service (alias for start)
  start                install if missing and start the service; show status
  stop                 stop the service but leave installation intact
  reload               stop then start again (ensures service installed)
  uninstall            remove the service completely (cleanup)
  status               show current service status and config info
  regenerate-key       generate new API key (writes config + reload)
  set-allowed-ip-hosts [ips]  configure allowed hosts (comma separated list)
  help                 show this message
EOF
    exit 1
}

ensure_config() {
    check_openssl
    if [ ! -f "$CONFIG" ]; then
        local newkey
        newkey=$(openssl rand -base64 24 | tr '+/' '-_' | tr -d '=')
        cat >"$CONFIG" <<EOF
api_key=$newkey
allowed_ips=127.0.0.1
EOF
        chmod 600 "$CONFIG"
        echo "created default config at $CONFIG"
    fi
}

show_info() {
    [ -f "$CONFIG" ] || return
    local api allowed
    api=$(grep '^api_key' "$CONFIG" | cut -d'=' -f2- | tr -d ' ')
    allowed=$(grep '^allowed_ips' "$CONFIG" | cut -d'=' -f2- | tr -d ' ')
    echo "api_key: $api"
    echo "allowed_ips: $allowed"
    echo "port: 8888"
}

regen_key() {
    ensure_config
    local newkey
    newkey=$(openssl rand -base64 24 | tr '+/' '-_' | tr -d '=')
    # replace or append api_key line
    if grep -q '^api_key=' "$CONFIG"; then
        sed -i "s|^api_key=.*|api_key=$newkey|" "$CONFIG"
    else
        echo "api_key=$newkey" >> "$CONFIG"
    fi
    echo "new API key written to $CONFIG"
    reload
}

set_allowed_ip_hosts() {
    ensure_config
    local input="$1"
    if [ -z "$input" ]; then
        read -rp "allowed IPs (comma separated): " input
    fi
    IFS=',' read -r -a userips <<<"$input"
    local known_ips=("127.0.0.1")
    local hostip
    hostip=$(hostname -I | awk '{print $1}')
    [ -n "$hostip" ] && known_ips+=("$hostip")
    # typical docker bridge
    known_ips+=("172.17.0.1")
    for ip in "${userips[@]}"; do
        [ -n "$ip" ] && known_ips+=("$ip")
    done
    # dedupe
    local all_ips=()
    for ip in "${known_ips[@]}"; do
        if ! printf '%s\n' "${all_ips[@]}" | grep -qx "$ip"; then
            all_ips+=("$ip")
        fi
    done
    # combine into comma list
    local newlist
    IFS=',' newlist="${all_ips[*]}"
    if grep -q '^allowed_ips=' "$CONFIG"; then
        sed -i "s|^allowed_ips=.*|allowed_ips=$newlist|" "$CONFIG"
    else
        echo "allowed_ips=$newlist" >> "$CONFIG"
    fi
    echo "updated allowed_ips in $CONFIG"
    reload
}

install_service() {
    ensure_config
    cat >/etc/systemd/system/$SERVICE_NAME <<EOF
[Unit]
Description=Retaliq domain helper
After=network.target

[Service]
Type=simple
ExecStart=$BIN -config $CONFIG
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    echo "service installed"
}

uninstall() {
    # stop first
    stop
    # disable and remove unit
    systemctl disable --now $SERVICE_NAME || true
    rm -f /etc/systemd/system/$SERVICE_NAME
    systemctl daemon-reload || true
    # kill stray processes
    pkill -f "$(basename "$BIN")" || true
    # remove config if present
    rm -f "$CONFIG" || true
    echo "uninstalled"
}

start() {
    # ensure service is installed, then start
    if ! systemctl list-unit-files | grep -q "^$SERVICE_NAME"; then
        install_service
    fi
    systemctl start $SERVICE_NAME || true
    show_info
}

status() {
    # show systemd status if unit exists (enabled/disabled/running)
    if systemctl status "$SERVICE_NAME" >/dev/null 2>&1; then
        systemctl status "$SERVICE_NAME" --no-pager
    else
        echo "$SERVICE_NAME not installed or inactive"
    fi
    show_info
}

stop() {
    # just stop the service; leave unit file / installation in place
    systemctl stop $SERVICE_NAME || true

    # kill background pid if exists
    if [ -f /var/run/retaliq-domain.pid ]; then
        kill "$(cat /var/run/retaliq-domain.pid)" || true
        rm -f /var/run/retaliq-domain.pid
    fi

    echo "stopped"
}

reload() {
    stop
    start
}

# default if no arguments
if [ $# -eq 0 ]; then
    usage
fi

case "$1" in
    install) start ;;            # install is alias for start
    start) start ;;
    stop) stop ;;
    reload) reload ;;
    uninstall) uninstall ;;
    status) status ;;
    "regenerate-key") regen_key ;;
    "set-allowed-ip-hosts") shift; set_allowed_ip_hosts "$*" ;;
    help) usage ;;
    *) usage ;;
esac
