#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${GO_INTENT_ANALYZER_INSTALL_DIR:-${HOME}/.local/bin}"
CLI_PATH="${INSTALL_DIR}/go-intent-analyzer"
CONFIG_BASE="${XDG_CONFIG_HOME:-${HOME}/.config}"
CONFIG_DIR="${CONFIG_BASE}/go-intent-analyzer"
ENV_FILE="${GO_INTENT_ANALYZER_ENV_FILE:-${CONFIG_DIR}/env}"
VSIX_PATH="${ROOT_DIR}/dist/go-intent-analyzer.vsix"
TEMP_BINARY=""

cleanup() {
  if [[ -n "${TEMP_BINARY}" && -f "${TEMP_BINARY}" ]]; then
    rm -f -- "${TEMP_BINARY}"
  fi
}
trap cleanup EXIT

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

find_code() {
  if [[ -n "${GO_INTENT_ANALYZER_CODE_BIN:-}" ]]; then
    if [[ ! -x "${GO_INTENT_ANALYZER_CODE_BIN}" ]]; then
      echo "error: GO_INTENT_ANALYZER_CODE_BIN is not executable: ${GO_INTENT_ANALYZER_CODE_BIN}" >&2
      exit 1
    fi
    printf '%s\n' "${GO_INTENT_ANALYZER_CODE_BIN}"
    return
  fi
  if command -v code >/dev/null 2>&1; then
    command -v code
    return
  fi
  local mac_code="/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
  if [[ -x "${mac_code}" ]]; then
    printf '%s\n' "${mac_code}"
    return
  fi
  echo "error: VS Code CLI was not found. Install the 'code' command from VS Code or install Visual Studio Code in /Applications." >&2
  exit 1
}

write_api_key() {
  local key="$1"
  local env_dir
  env_dir="$(dirname "${ENV_FILE}")"
  mkdir -p -- "${env_dir}"
  umask 077
  local temp_env
  temp_env="$(mktemp "${env_dir}/env.XXXXXX")"
  printf 'TYPESAFE_API_KEY=%s\n' "${key}" > "${temp_env}"
  chmod 600 "${temp_env}"
  mv -f -- "${temp_env}" "${ENV_FILE}"
}

require_command go
require_command pnpm
CODE_BIN="$(find_code)"

echo "[1/4] Building Go CLI..."
mkdir -p -- "${INSTALL_DIR}"
TEMP_BINARY="$(mktemp "${INSTALL_DIR}/go-intent-analyzer.XXXXXX")"
(cd "${ROOT_DIR}" && go build -o "${TEMP_BINARY}" ./cmd)
chmod 755 "${TEMP_BINARY}"
mv -f -- "${TEMP_BINARY}" "${CLI_PATH}"
TEMP_BINARY=""

echo "[2/4] Building VS Code extension..."
pnpm --dir "${ROOT_DIR}/editors/vscode" install --frozen-lockfile
pnpm --dir "${ROOT_DIR}/editors/vscode" run package
if [[ ! -f "${VSIX_PATH}" ]]; then
  echo "error: VSIX was not generated: ${VSIX_PATH}" >&2
  exit 1
fi

echo "[3/4] Installing VS Code extension..."
"${CODE_BIN}" --install-extension "${VSIX_PATH}" --force

echo "[4/4] Configuring TypeSafe API key..."
if [[ -n "${TYPESAFE_API_KEY:-}" ]]; then
  write_api_key "${TYPESAFE_API_KEY}"
elif [[ -f "${ENV_FILE}" ]] && grep -q '^TYPESAFE_API_KEY=.' "${ENV_FILE}"; then
  chmod 600 "${ENV_FILE}"
  echo "Existing API key configuration retained."
elif [[ -t 0 || -r /dev/tty ]]; then
  read -r -s -p "TypeSafe API key: " api_key </dev/tty
  printf '\n' >/dev/tty
  if [[ -z "${api_key}" ]]; then
    echo "error: API key cannot be empty" >&2
    exit 1
  fi
  write_api_key "${api_key}"
else
  echo "error: TYPESAFE_API_KEY is not set and no interactive terminal is available" >&2
  exit 1
fi

echo
echo "Setup complete."
echo "CLI: ${CLI_PATH}"
echo "VSIX: ${VSIX_PATH}"
echo "API key file: ${ENV_FILE} (mode 600)"
echo
echo "Reload VS Code, open a Go workspace, and run:"
echo "  Go Intent Analyzer: Analyze Workspace"
