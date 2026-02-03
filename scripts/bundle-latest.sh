#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "${ROOT_DIR}"

VERSION_FILE="zid-logs-latest.version"
if [ ! -f "${VERSION_FILE}" ]; then
	echo "ERROR: ${VERSION_FILE} not found" >&2
	exit 2
fi
VERSION="$(head -n 1 "${VERSION_FILE}" | tr -d '\r' | tr -d '\n')"
if [ -z "${VERSION}" ]; then
	echo "ERROR: ${VERSION_FILE} is empty" >&2
	exit 2
fi

if [ ! -f build/zid-logs ]; then
	echo "ERROR: missing binary in ./build. Run: make build" >&2
	exit 2
fi

STAGE_BASE="dist"
STAGE_DIR_PFSENSE="${STAGE_BASE}/zid-logs-pfsense"

rm -rf "${STAGE_DIR_PFSENSE}"
mkdir -p "${STAGE_DIR_PFSENSE}/build"

cp -R packaging/pfsense/pkg-zid-logs "${STAGE_DIR_PFSENSE}/pkg-zid-logs"

cp -f build/zid-logs "${STAGE_DIR_PFSENSE}/build/zid-logs"
chmod 755 "${STAGE_DIR_PFSENSE}/build/zid-logs"

printf "%s\n" "${VERSION}" > "${STAGE_DIR_PFSENSE}/VERSION"

OUT_PFSENSE="zid-logs-latest.tar.gz"

bundle_one() {
	src_dir="$1"
	out="$2"
	tmp_out="${out}.tmp.$$"
	rm -f "${tmp_out}"
	tar -czf "${tmp_out}" -C "${STAGE_BASE}" "${src_dir}"
	mv -f "${tmp_out}" "${out}"
}

bundle_one "zid-logs-pfsense" "${OUT_PFSENSE}"

hash_one() {
	out="$1"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "${out}" | awk '{print $1}'
		return 0
	elif command -v sha256 >/dev/null 2>&1; then
		sha256 -q "${out}"
		return 0
	fi
	return 1
}

TMP_SHA="$(mktemp)"
if [ -f "${OUT_PFSENSE}" ]; then
	HASH="$(hash_one "${OUT_PFSENSE}" || true)"
	if [ -n "${HASH}" ]; then
		printf "%s  %s\n" "${HASH}" "${OUT_PFSENSE}" >> "${TMP_SHA}"
	else
		echo "WARN: could not compute sha256 for ${OUT_PFSENSE}" >&2
	fi
fi
mv -f "${TMP_SHA}" sha256.txt

ls -lh "${OUT_PFSENSE}" sha256.txt
