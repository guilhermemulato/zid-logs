#!/bin/sh
#
# zid-logs-update (bootstrap)
#
# Small, stable updater wrapper that always runs the updater shipped in the latest bundle.
#
# Usage:
#   sh /usr/local/sbin/zid-logs-update
#   sh /usr/local/sbin/zid-logs-update -u https://.../zid-logs-latest.tar.gz
#

set -eu

URL_DEFAULT="https://s3.soulsolucoes.com.br/soul/portal/zid-logs-latest.tar.gz"
URL="${ZID_LOGS_UPDATE_URL:-$URL_DEFAULT}"
FORCE=0
KEEP_TMP=0

usage() {
	cat <<EOF
ZID Logs updater (bootstrap)

Usage:
  sh /usr/local/sbin/zid-logs-update [-u <url>] [-f] [-k]

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

UPDATER_SH="$(find "${EXTRACT_DIR}" -maxdepth 5 -type f -path "*/pkg-zid-logs/update.sh" | head -n 1 || true)"
if [ -z "${UPDATER_SH}" ]; then
	die "update.sh not found inside bundle (expected */pkg-zid-logs/update.sh)"
fi

echo "Running bundled updater: ${UPDATER_SH}"
echo ""

UPDATER_ARGS=""
if [ "${KEEP_TMP}" -eq 1 ]; then
	UPDATER_ARGS="${UPDATER_ARGS} -k"
fi
if [ "${URL}" != "${URL_DEFAULT}" ]; then
	UPDATER_ARGS="${UPDATER_ARGS} -u ${URL}"
fi
if [ "${FORCE}" -eq 1 ]; then
	UPDATER_ARGS="${UPDATER_ARGS} -f"
fi

sh "${UPDATER_SH}" ${UPDATER_ARGS}
