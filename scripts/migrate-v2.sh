#!/bin/sh
# Uses the native binary only. Default mode previews; --apply commits a backup-backed import.
set -eu
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ -n "${VSC_BINARY:-}" ]; then
  binary=$VSC_BINARY
elif [ -x "$script_dir/../vsc" ]; then
  binary=$script_dir/../vsc
elif [ "$(uname -s)" = Darwin ] && [ -x "$script_dir/../build/vsc" ]; then
  binary=$script_dir/../build/vsc
elif [ "$(uname -s)" = Linux ] && [ -x "$script_dir/../build/vsc-linux-amd64" ]; then
  binary=$script_dir/../build/vsc-linux-amd64
else
  printf '%s\n' '未找到新版二进制，请设置 VSC_BINARY 为 vsc 可执行文件的完整路径。' >&2
  exit 1
fi
exec "$binary" migrate-legacy "$@"
