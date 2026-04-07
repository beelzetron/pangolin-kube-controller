#!/usr/bin/env bash
set -euo pipefail

# Update vendored Traefik CRDs for a given tag.
# Usage: TRAEFIK_TAG=v3.5.1 hack/scripts/update-traefik-crds.sh

TRAEFIK_TAG="${TRAEFIK_TAG:-v3.5.1}"
OUT_DIR="${OUT_DIR:-test/crds/traefik/${TRAEFIK_TAG#v}}"
mkdir -p "$OUT_DIR"

# Sources: using Traefik Helm chart CRDs at the specified tag.
# Adjust if upstream layout changes in the future.
base_url="https://raw.githubusercontent.com/traefik/traefik/${TRAEFIK_TAG}/docs/content/reference/dynamic-configuration"

# Download the bundled CRD manifest (single file contains all Traefik CRDs)
declare -A FILES=(
	["traefik-crds.yaml"]="${base_url}/kubernetes-crd-definition-v1.yml"
)

# Also optionally download individual Traefik CRD files (traefik.io_*.yaml)
# This mirrors older layouts where each CRD was provided as a separate file.
# Set DOWNLOAD_SPLIT=false to skip fetching these.
DOWNLOAD_SPLIT="${DOWNLOAD_SPLIT:-true}"
INDIVIDUAL_FILES=(
	"traefik.io_ingressroutes.yaml"
	"traefik.io_ingressroutetcps.yaml"
	"traefik.io_ingressrouteudps.yaml"
	"traefik.io_middlewares.yaml"
	"traefik.io_middlewaretcps.yaml"
	"traefik.io_serverstransports.yaml"
	"traefik.io_serverstransporttcps.yaml"
	"traefik.io_traefikservices.yaml"
	"traefik.io_tlsoptions.yaml"
	"traefik.io_tlsstores.yaml"
)

sha256() {
	local file="$1"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$file" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$file" | awk '{print $1}'
	else
		# OpenSSL output formats vary; parse robustly by splitting on "= " and
		# returning the last field which is the hex digest in known formats.
		openssl dgst -sha256 "$file" | awk -F'= ' '{print $NF}'
	fi
	return 0
}

printf "Sync Traefik CRDs for %s -> %s\n" "$TRAEFIK_TAG" "$OUT_DIR"

for fname in "${!FILES[@]}"; do
	url="${FILES[$fname]}"
	tmp="$(mktemp)"
	echo "Downloading $url"
	curl -fsSL "$url" -o "$tmp"
	# Normalize to LF
	tr -d '\r' <"$tmp" >"$OUT_DIR/$fname"
	rm -f "$tmp"
	sum="$(sha256 "$OUT_DIR/$fname")"
	printf "%-35s %10d bytes  sha256:%s\n" "$fname" "$(wc -c <"$OUT_DIR/$fname")" "$sum"
done

if [[ $DOWNLOAD_SPLIT == "true" ]]; then
	for fname in "${INDIVIDUAL_FILES[@]}"; do
		url="${base_url}/${fname}"
		tmp="$(mktemp)"
		echo "Downloading $url"
		if curl -fsSL "$url" -o "$tmp"; then
			# Normalize to LF
			tr -d '\r' <"$tmp" >"$OUT_DIR/$fname"
			rm -f "$tmp"
			sum="$(sha256 "$OUT_DIR/$fname")"
			printf "%-35s %10d bytes  sha256:%s\n" "$fname" "$(wc -c <"$OUT_DIR/$fname")" "$sum"
		else
			# upstream may not provide every individual file for every tag
			echo "Warning: $fname not found at $url (skipping)" >&2
			rm -f "$tmp"
		fi
	done
fi

echo "Done. Review changes and adjust URLs if upstream moved files."
