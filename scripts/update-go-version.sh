#!/usr/bin/env bash

set -euo pipefail

readonly VERSION_INPUT="${1:-}"

log() {
    echo "[update-go-version] $*"
}

die() {
    echo "[update-go-version] ERROR: $*" >&2
    exit 1
}

require_file() {
    local file="$1"
    [[ -f "$file" ]] || die "Required file not found: $file"
}

require_cmd() {
    local cmd="$1"
    command -v "$cmd" > /dev/null 2>&1 || die "Required command not found: $cmd"
}

validate_version() {
    [[ "$VERSION_INPUT" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "Invalid Go version '$VERSION_INPUT'. Expected format: MAJOR.MINOR.PATCH"
}

replace_or_fail() {
    local file="$1"
    local match_regex="$2"
    local sed_expr="$3"

    grep -Eq "$match_regex" "$file" || die "Expected pattern not found in $file"
    sed -E -i "$sed_expr" "$file"
}

update_known_version_files() {
    log "Updating Go version references in known files"

    replace_or_fail "go.mod" '^go [0-9]+\.[0-9]+\.[0-9]+$' "s/^go [0-9]+\.[0-9]+\.[0-9]+$/go ${VERSION_INPUT}/"
    replace_or_fail "go.tool.mod" '^go [0-9]+\.[0-9]+\.[0-9]+$' "s/^go [0-9]+\.[0-9]+\.[0-9]+$/go ${VERSION_INPUT}/"
    replace_or_fail ".golangci.yaml" '^[[:space:]]*go:[[:space:]]*"?[0-9]+\.[0-9]+(\.[0-9]+)?"?[[:space:]]*$' "s/^([[:space:]]*go:[[:space:]]*).*/\\1\"${VERSION_INPUT}\"/"

    require_file ".mise.toml"
    awk -v version="$VERSION_INPUT" '
    BEGIN { in_tools = 0; updated = 0 }
    {
      if ($0 ~ /^\[tools\][[:space:]]*$/) {
        in_tools = 1
        print
        next
      }
      if (in_tools && $0 ~ /^\[[^]]+\][[:space:]]*$/) {
        in_tools = 0
      }
      if (in_tools && !updated && $0 ~ /^[[:space:]]*(go|golang)[[:space:]]*=[[:space:]]*"[^"]+"[[:space:]]*$/) {
        sub(/"[^"]+"/, "\"" version "\"")
        updated = 1
      }
      print
    }
    END {
      if (!updated) {
        print "missing go/golang entry in [tools] section of .mise.toml; add one like: [tools] go = \"" version "\"" > "/dev/stderr"
        exit 1
      }
    }
  ' .mise.toml > .mise.toml.tmp && mv .mise.toml.tmp .mise.toml

    if [[ -f "mise.toml" ]]; then
        log "Updating optional mise.toml"
        sed -E -i "s/^([[:space:]]*(go|golang)[[:space:]]*=[[:space:]]*)\"[^\"]+\"[[:space:]]*$/\\1\"${VERSION_INPUT}\"/" mise.toml || true
    fi
}

show_detected_go_version_refs() {
    log "Scanning tracked files for Go version keys"
    grep -HnE '(^go [0-9]+\.[0-9]+\.[0-9]+$|^[[:space:]]*go:[[:space:]]*"?[0-9]+\.[0-9]+(\.[0-9]+)?"?[[:space:]]*$|^[[:space:]]*(go|golang)[[:space:]]*=[[:space:]]*"[^"]+"[[:space:]]*$)' go.mod go.tool.mod .golangci.yaml .mise.toml 2> /dev/null || true
    if [[ -f "mise.toml" ]]; then
        grep -HnE '^[[:space:]]*(go|golang)[[:space:]]*=[[:space:]]*"[^"]+"[[:space:]]*$' mise.toml 2> /dev/null || true
    fi
}

main() {
    validate_version
    require_file "go.mod"
    require_file "go.tool.mod"
    require_file ".golangci.yaml"
    require_file ".mise.toml"
    require_cmd grep
    require_cmd sed
    require_cmd awk
    require_cmd curl
    require_cmd docker

    show_detected_go_version_refs
    update_known_version_files
    log "Done. Updated Go references to $VERSION_INPUT"
}

main
