#!/bin/sh
#
# update.sh
#
# Downloads and applies the latest ZID Logs pfSense bundle.
# - Fetches a tar.gz from the URL below (override with -u or $ZID_LOGS_UPDATE_URL)
# - Extracts to a temporary directory
# - Runs the bundled pkg-zid-logs/install.sh
#
# Usage:
#   sh update.sh
#   sh update.sh -u https://.../zid-logs-latest.tar.gz
#   ZID_LOGS_UPDATE_URL=... sh update.sh
#

set -eu

URL_DEFAULT="https://s3.soulsolucoes.com.br/soul/portal/zid-logs-latest.tar.gz"
URL="${ZID_LOGS_UPDATE_URL:-$URL_DEFAULT}"
KEEP_TMP=0
WAS_RUNNING=0
FORCE=0
WAS_ENABLED=0

usage() {
	cat <<EOF
ZID Logs updater

Usage:
  sh update.sh [-u <url>] [-f] [-k]

Options:
  -u <url>  Bundle URL (default: ${URL_DEFAULT})
  -f        Force update (skip version check)
  -k        Keep temporary directory (debug)
EOF
}

die() {
	echo "ERROR: $*" >&2
	exit 1
}

sha256_file() {
	if command -v sha256 >/dev/null 2>&1; then
		sha256 -q "$1" 2>/dev/null || true
	fi
}

pids() {
	if command -v pgrep >/dev/null 2>&1; then
		pgrep -f '/usr/local/sbin/zid-logs' 2>/dev/null || true
		return
	fi
	ps ax -o pid= -o command= | awk '/\/usr\/local\/sbin\/zid-logs/ {print $1}'
}

is_enabled() {
	if [ -f /usr/local/etc/zid-logs/config.json ] && command -v jq >/dev/null 2>&1; then
		jq -r '.enabled // false' /usr/local/etc/zid-logs/config.json 2>/dev/null | grep -qi '^true$'
		return $?
	fi
	if [ -f /usr/local/etc/zid-logs/config.json ]; then
		grep -qi '"enabled"[[:space:]]*:[[:space:]]*true' /usr/local/etc/zid-logs/config.json
		return $?
	fi
	return 1
}

set_rc_enable() {
	value="$1"
	conf="/etc/rc.conf.local"
	line="zid_logs_enable=\"${value}\""
	if [ ! -f "${conf}" ]; then
		touch "${conf}"
	fi
	if grep -q '^zid_logs_enable=' "${conf}"; then
		sed -i.bak "s/^zid_logs_enable=.*/${line}/" "${conf}"
	else
		echo "${line}" >> "${conf}"
	fi
	chmod 0644 "${conf}" 2>/dev/null || true
}

while getopts "u:fkh" opt; do
	case "$opt" in
		u) URL="$OPTARG" ;;
		f) FORCE=1 ;;
		k) KEEP_TMP=1 ;;
		h) usage; exit 0 ;;
		*) usage; exit 2 ;;
	esac
done

if [ "$(id -u)" != "0" ]; then
	die "This script must be run as root"
fi

if ! command -v tar >/dev/null 2>&1; then
	die "tar not found"
fi

DOWNLOADER=""
if command -v fetch >/dev/null 2>&1; then
	DOWNLOADER="fetch"
elif command -v curl >/dev/null 2>&1; then
	DOWNLOADER="curl"
else
	die "Neither 'fetch' nor 'curl' found (pfSense usually provides 'fetch')"
fi

get_local_version() {
	if [ -x /usr/local/sbin/zid-logs ]; then
		/usr/local/sbin/zid-logs -version 2>/dev/null | awk '{print $3}' | head -n 1 | tr -d '\r'
	fi
}

get_remote_version() {
	version_url="$1"
	if [ "${DOWNLOADER}" = "fetch" ]; then
		fetch -q -o - "${version_url}" 2>/dev/null | head -n 1 | tr -d '\r'
	else
		curl -fsSL "${version_url}" 2>/dev/null | head -n 1 | tr -d '\r'
	fi
}

version_url="${URL}"
case "${version_url}" in
	*.tar.gz) version_url="${version_url%.tar.gz}.version" ;;
	*.tgz) version_url="${version_url%.tgz}.version" ;;
	*) version_url="${version_url}.version" ;;
esac

if [ "${FORCE}" -eq 0 ]; then
	local_version="$(get_local_version || true)"
	remote_version="$(get_remote_version "${version_url}" || true)"
	if [ -n "${remote_version}" ] && [ -n "${local_version}" ] && [ "${remote_version}" = "${local_version}" ]; then
		echo "Already up-to-date (version ${local_version})."
		exit 0
	fi
fi

TMP_DIR="$(mktemp -d /tmp/zid-logs-update.XXXXXX)"
cleanup() {
	if [ "${KEEP_TMP}" -eq 1 ]; then
		echo "Keeping temp dir: ${TMP_DIR}"
		return
	fi
	rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

stop_all() {
	echo "Stopping service (best-effort)..."
	PIDS="$(pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
	if [ -n "${PIDS}" ]; then
		echo "Stopping running processes: ${PIDS}"
		kill ${PIDS} 2>/dev/null || true

		i=0
		while [ $i -lt 10 ]; do
			PIDS_NOW="$(pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
			if [ -z "${PIDS_NOW}" ]; then
				break
			fi
			sleep 1
			i=$((i + 1))
		done

		PIDS_NOW="$(pids | tr '\n' ' ' | sed 's/[[:space:]]*$//')"
		if [ -n "${PIDS_NOW}" ]; then
			echo "Processes still running; sending SIGKILL: ${PIDS_NOW}"
			kill -9 ${PIDS_NOW} 2>/dev/null || true
			sleep 1
		fi
	fi

	if [ -f /var/run/zid-logs.pid ]; then
		PID="$(cat /var/run/zid-logs.pid 2>/dev/null || true)"
		if [ -z "${PID}" ] || ! kill -0 "${PID}" 2>/dev/null; then
			rm -f /var/run/zid-logs.pid 2>/dev/null || true
		fi
	fi
}

TARBALL="${TMP_DIR}/bundle.tar.gz"
EXTRACT_DIR="${TMP_DIR}/extract"
mkdir -p "${EXTRACT_DIR}"

echo "========================================="
echo " ZID Logs Update"
echo "========================================="
echo ""
echo "Downloading: ${URL}"

if [ "${DOWNLOADER}" = "fetch" ]; then
	fetch -o "${TARBALL}" "${URL}"
else
	curl -fL -o "${TARBALL}" "${URL}"
fi

echo "Extracting bundle..."
tar -xzf "${TARBALL}" -C "${EXTRACT_DIR}"

INSTALL_SH="$(find "${EXTRACT_DIR}" -maxdepth 5 -type f -path "*/pkg-zid-logs/install.sh" | head -n 1 || true)"
if [ -z "${INSTALL_SH}" ]; then
	die "install.sh not found inside bundle (expected */pkg-zid-logs/install.sh)"
fi

PKG_DIR="$(dirname "${INSTALL_SH}")"

echo ""
echo "Bundle verification:"
if [ -f "${PKG_DIR}/files/usr/local/www/zid-logs_config.php" ]; then
	HASH_SRC="$(sha256_file "${PKG_DIR}/files/usr/local/www/zid-logs_config.php")"
	if [ -n "${HASH_SRC}" ]; then
		echo "  src zid-logs_config.php sha256: ${HASH_SRC}"
	fi
fi

if [ -n "$(pids | head -n 1)" ]; then
	WAS_RUNNING=1
fi
if is_enabled; then
	WAS_ENABLED=1
fi

if [ "${WAS_ENABLED}" -eq 1 ]; then
	set_rc_enable "YES"
else
	set_rc_enable "NO"
fi

stop_all

echo ""
echo "Applying update from: ${PKG_DIR}"
sh "${INSTALL_SH}"

if [ -f /usr/local/www/zid-logs_config.php ]; then
	HASH_DST="$(sha256_file /usr/local/www/zid-logs_config.php)"
	if [ -n "${HASH_DST}" ]; then
		echo ""
		echo "Installed verification:"
		echo "  dst zid-logs_config.php sha256: ${HASH_DST}"
	fi
fi

echo ""
echo "Restarting service..."
if [ "${WAS_RUNNING}" -eq 1 ] || [ "${WAS_ENABLED}" -eq 1 ]; then
	stop_all
	if [ -x /usr/local/etc/rc.d/zid_logs ]; then
		/usr/local/etc/rc.d/zid_logs onestart 2>/dev/null || true
	else
		service zid_logs onestart 2>/dev/null || true
	fi
else
	echo "(Service was not running before update; not forcing start.)"
fi

if [ "${ZID_LOGS_SKIP_WEBGUI_RELOAD:-0}" -eq 0 ]; then
	echo ""
	echo "Reloading pfSense web GUI (to pick up updated PHP pages)..."
	if [ -x /usr/local/sbin/pfSsh.php ]; then
		/usr/local/sbin/pfSsh.php playback reloadwebgui >/dev/null 2>&1 || true
	elif [ -x /etc/rc.restart_webgui ]; then
		/etc/rc.restart_webgui >/dev/null 2>&1 || true
	elif [ -x /usr/local/etc/rc.d/php-fpm ]; then
		/usr/local/etc/rc.d/php-fpm restart >/dev/null 2>&1 || true
	fi
else
	echo ""
	echo "Skipping web GUI reload (ZID_LOGS_SKIP_WEBGUI_RELOAD=1)."
fi

echo ""
echo "========================================="
echo " Update Complete"
echo "========================================="
echo ""
