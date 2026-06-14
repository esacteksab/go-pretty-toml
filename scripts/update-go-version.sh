#!/usr/bin/env bash

set -euo pipefail

readonly VERSION_INPUT="${1:-}"
readonly DOCKERFILE="Dockerfile"

# Global initialization for Docker Hub authentication token
HUB_TOKEN=""

# Manage temporary files robustly across execution lifecycles
declare -a CLEANUP_FILES=()
cleanup() {
  if [[ ${#CLEANUP_FILES[@]} -gt 0 ]]; then
    rm -f "${CLEANUP_FILES[@]}"
  fi
}
trap cleanup EXIT

log() {
  echo "[update-go-version] $*" >&2
}

die() {
  echo "[update-go-version] ERROR: $*" >&2
  exit 1
}

require_cmd() {
  local cmd="$1"
  command -v "$cmd" >/dev/null 2>&1 || die "Required command not found: $cmd"
}

validate_version() {
  [[ "$VERSION_INPUT" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "Invalid Go version '$VERSION_INPUT'. Expected format: MAJOR.MINOR.PATCH"
}

replace_or_fail() {
  local file="$1"
  local match_regex="$2"
  local sed_expr="$3"

  grep -Eq "$match_regex" "$file" || die "Expected version pattern layout not found in existing file: $file"
  sed -E -i "$sed_expr" "$file"
}

hub_authenticate() {
  if [[ -n "${DOCKERHUB_USERNAME:-}" && -n "${DOCKERHUB_TOKEN:-}" ]]; then
    log "Authenticating to Docker Hub as ${DOCKERHUB_USERNAME}"
    local resp

    resp="$(curl -sS \
      -H 'Content-Type: application/json' \
      -d "{\"username\": \"${DOCKERHUB_USERNAME}\", \"password\": \"${DOCKERHUB_TOKEN}\"}" \
      "https://hub.docker.com/v2/users/login" 2>/dev/null || true)"

    HUB_TOKEN="$(printf '%s' "$resp" | jq -r '.token // empty')"
    [[ -n "$HUB_TOKEN" ]] || die "Docker Hub authentication failed (check credentials)."
    log "Docker Hub authentication succeeded"
  else
    log "No DOCKERHUB_USERNAME/DOCKERHUB_TOKEN set; using anonymous Hub API access"
  fi
}

hub_get() {
  local url="$1"
  local attempt=1
  local max_attempts="${HUB_MAX_ATTEMPTS:-6}"
  local delay=2

  while :; do
    local body_file
    body_file="$(mktemp)"
    CLEANUP_FILES+=("$body_file")

    local headers_file
    headers_file="$(mktemp)"
    CLEANUP_FILES+=("$headers_file")

    local curl_args=(
      -sS
      -o "$body_file"
      -D "$headers_file"
      -w '%{http_code}'
      -H 'Accept: application/json'
    )

    if [[ -n "${HUB_TOKEN}" ]]; then
      curl_args+=( -H "Authorization: Bearer ${HUB_TOKEN}" )
    fi

    local http
    http="$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo "000")"

    if [[ "$http" == "200" ]]; then
      cat "$body_file"
      return 0
    fi

    if { [[ "$http" == "429" ]] || [[ "$http" == 5* ]] || [[ "$http" == "000" ]]; } && (( attempt < max_attempts )); then
      local retry_after
      retry_after="$(grep -i '^Retry-After:' "$headers_file" 2>/dev/null | tail -n1 \
        | sed -E 's/^[Rr]etry-[Aa]fter:[[:space:]]*([0-9]+).*/\1/' | tr -d '\r')"

      local sleep_for
      if [[ "$retry_after" =~ ^[0-9]+$ ]]; then
        sleep_for="$retry_after"
      else
        sleep_for="$delay"
      fi

      log "Docker Hub API returned HTTP ${http}; retry ${attempt}/${max_attempts} after ${sleep_for}s"
      sleep "$sleep_for"
      (( delay = delay * 2 > 60 ? 60 : delay * 2 ))
      (( ++attempt ))
      continue
    fi

    die "Docker Hub request failed (HTTP ${http}): ${url}"
  done
}

update_known_version_files() {
  log "Updating Go version references in available files..."

  if [[ -f "go.mod" ]]; then
    log "Processing go.mod..."
    replace_or_fail "go.mod" '^go [0-9]+\.[0-9]+\.[0-9]+$' "s/^go [0-9]+\.[0-9]+\.[0-9]+$/go ${VERSION_INPUT}/"
  fi

  if [[ -f "go.tool.mod" ]]; then
    log "Processing go.tool.mod..."
    replace_or_fail "go.tool.mod" '^go [0-9]+\.[0-9]+\.[0-9]+$' "s/^go [0-9]+\.[0-9]+\.[0-9]+$/go ${VERSION_INPUT}/"
  fi

  if [[ -f ".golangci.yaml" ]]; then
    log "Processing .golangci.yaml..."
    replace_or_fail ".golangci.yaml" '^[[:space:]]*go:[[:space:]]*"?[0-9]+\.[0-9]+(\.[0-9]+)?"?[[:space:]]*$' "s/^([[:space:]]*go:[[:space:]]*).*/\\1\"${VERSION_INPUT}\"/"
  fi

  if [[ -f ".mise.toml" ]]; then
    log "Processing .mise.toml..."
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
          print "missing go/golang entry in [tools] section of .mise.toml" > "/dev/stderr"
          exit 1
        }
      }
    ' .mise.toml > .mise.toml.tmp && mv .mise.toml.tmp .mise.toml
  fi

  if [[ -f "mise.toml" ]]; then
    log "Processing optional mise.toml..."
    sed -E -i "s/^([[:space:]]*(go|golang)[[:space:]]*=[[:space:]]*)\"[^\"]+\"[[:space:]]*$/\\1\"${VERSION_INPUT}\"/" mise.toml || true
  fi
}

extract_repo_and_stage_from_dockerfile() {
  local from_line
  from_line="$(awk '/^FROM[[:space:]]+/ { print; exit }' "$DOCKERFILE")"
  [[ -n "$from_line" ]] || die "No FROM line found in $DOCKERFILE"

  local image_ref
  image_ref="$(printf '%s\n' "$from_line" | sed -E 's/^FROM[[:space:]]+([^[:space:]]+).*/\1/')"
  local stage_suffix
  stage_suffix="$(printf '%s\n' "$from_line" | sed -nE 's/^FROM[[:space:]]+[^[:space:]]+([[:space:]]+AS[[:space:]]+.+)$/\1/p')"

  [[ "$image_ref" =~ ^([^@]+)@sha256:[a-f0-9]{64}$ ]] || die "Expected digest-pinned base image in $DOCKERFILE"
  local image_with_tag="${BASH_REMATCH[1]}"
  local repo="${image_with_tag%:*}"
  [[ "$repo" != "$image_with_tag" ]] || die "Unable to parse repo and tag from Dockerfile FROM line"

  printf '%s\t%s\n' "$repo" "$stage_suffix"
}

find_latest_dated_tag() {
  local repo="$1"
  local ns_repo="$repo"
  if [[ "$ns_repo" =~ ^docker\.io/(.+/.+)$ ]]; then
    ns_repo="${BASH_REMATCH[1]}"
  fi
  [[ "$ns_repo" =~ ^[^./]+/[^/]+$ ]] || die "Docker Hub lookup expects repo in namespace/name form, got: $repo"

  local namespace="${ns_repo%/*}"
  local repository="${ns_repo#*/}"

  local api_url="https://hub.docker.com/v2/namespaces/${namespace}/repositories/${repository}/tags?page_size=100&name=${VERSION_INPUT}-"
  local matches=""

  local safe_version="${VERSION_INPUT//./\\.}"

  while [[ -n "$api_url" && "$api_url" != "null" ]]; do
    local body
    body="$(hub_get "$api_url")"

    local page
    page="$(printf '%s\n' "$body" | jq -r --arg regex "^${safe_version}-[0-9]{4}-[0-9]{2}-[0-9]{2}$" '
      .results[].name | select(. != null and test($regex))
    ')"

    if [[ -n "$page" ]]; then
      matches+=$'\n'
      matches+="$page"
    fi

    api_url="$(printf '%s\n' "$body" | jq -r '.next // empty')"
  done

  local latest
  latest="$(printf '%s\n' "$matches" | sed '/^[[:space:]]*$/d' | sort -u | tail -n 1)"
  [[ -n "$latest" ]] || die "No Docker Hub dated tag found for ${repo} with version ${VERSION_INPUT}"
  printf '%s\n' "$latest"
}

resolve_digest_with_docker() {
  local image_ref="$1"
  local digest=""

  local err_file
  err_file="$(mktemp)"
  CLEANUP_FILES+=("$err_file")

  export DOCKER_CONTENT_TRUST="${DOCKER_CONTENT_TRUST:-1}"

  local inspect_output
  if inspect_output="$(docker buildx imagetools inspect "$image_ref" 2>"$err_file")"; then
    if [[ "$inspect_output" =~ Digest:[[:space:]]*(sha256:[a-f0-9]{64}) ]]; then
      digest="${BASH_REMATCH[1]}"
    fi
  else
    log "Notice: imagetools inspect failed (stderr: $(cat "$err_file")). Falling back to docker pull."
  fi

  if [[ -z "$digest" ]]; then
    if docker pull "$image_ref" >/dev/null 2>"$err_file"; then
      local repo_digest
      repo_digest="$(docker image inspect --format '{{index .RepoDigests 0}}' "$image_ref" 2>/dev/null || true)"
      if [[ "$repo_digest" == *@sha256:* ]]; then
        digest="${repo_digest##*@}"
      fi
    else
      log "Notice: docker pull failed (stderr: $(cat "$err_file"))."
    fi
  fi

  [[ "$digest" =~ ^sha256:[a-f0-9]{64}$ ]] || die "Failed to resolve digest for $image_ref via docker"
  printf '%s\n' "$digest"
}

resolve_base_image() {
  local parsed
  parsed="$(extract_repo_and_stage_from_dockerfile)"

  local repo="${parsed%%$'\t'*}"
  local stage_suffix="${parsed#*$'\t'}"

  local tag
  tag="$(find_latest_dated_tag "$repo")"

  local image_ref="${repo}:${tag}"
  log "Resolving digest for $image_ref"

  local digest
  digest="$(resolve_digest_with_docker "$image_ref")"

  printf 'FROM %s@%s%s\n' "$image_ref" "$digest" "$stage_suffix"
}

update_dockerfile_base_image() {
  local replacement_line="$1"

  awk -v line="$replacement_line" '
    BEGIN { done = 0 }
    {
      if (!done && $0 ~ /^FROM[[:space:]]+/) {
        print line
        done = 1
      } else {
        print
      }
    }
    END {
      if (!done) { exit 1 }
    }' "$DOCKERFILE" > "$DOCKERFILE.tmp" && mv "$DOCKERFILE.tmp" "$DOCKERFILE"
}

show_detected_go_version_refs() {
  log "Scanning tracked files for Go version keys"

  local files=()
  for f in go.mod go.tool.mod .golangci.yaml .mise.toml mise.toml; do
    if [[ -f "$f" ]]; then
      files+=("$f")
    fi
  done

  if [[ ${#files[@]} -gt 0 ]]; then
    grep -HnE '(^go [0-9]+\.[0-9]+\.[0-9]+$|^[[:space:]]*go:[[:space:]]*"?[0-9]+\.[0-9]+(\.[0-9]+)?"?[[:space:]]*$|^[[:space:]]*(go|golang)[[:space:]]*=[[:space:]]*"[^"]+"[[:space:]]*$)' "${files[@]}" || true
  fi
}

main() {
  validate_version

  # Identify which known targets are explicitly present
  local targets_found=0

  for file in "go.mod" "go.tool.mod" ".golangci.yaml" ".mise.toml" "mise.toml" "$DOCKERFILE"; do
    if [[ -f "$file" ]]; then
      (( ++targets_found ))
    fi
  done

  if [[ "$targets_found" -eq 0 ]]; then
    die "No supported files (go.mod, go.tool.mod, .golangci.yaml, mise configs, or Dockerfile) found in this directory."
  fi

  # Require core utilities for standard file updates
  require_cmd grep
  require_cmd sed
  require_cmd awk

  # 1. Update text configurations
  show_detected_go_version_refs
  update_known_version_files

  # 2. Dynamic Container Base Image updates
  if [[ -f "$DOCKERFILE" ]]; then
    log "Dockerfile found. Updating container base image..."

    # Context-load tool requirements for container verification
    require_cmd curl
    require_cmd docker
    require_cmd jq
    require_cmd mktemp

    hub_authenticate

    local dockerfile_replacement_line
    dockerfile_replacement_line="$(resolve_base_image)"
    update_dockerfile_base_image "$dockerfile_replacement_line"
  fi

  log "Done. Project references successfully bumped to Go $VERSION_INPUT"
}

main "$@"
