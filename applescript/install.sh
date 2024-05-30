#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
pushd "$SCRIPT_DIR" || exit 1

./restore-resources.sh
mkdir -p "$HOME/Library/Scripts/Applications/Finder"
echo "installing scripts to $HOME/Library/Scripts/Applications/Finder ..."

INSTALL_DATE=$(date +%FT%T%Z)
COMMENT="installed $INSTALL_DATE"
XTOOL_VERSION=${XTOOL_VERSION:-""}
if [ -n "$XTOOL_VERSION" ]; then
  COMMENT="xtool $XTOOL_VERSION; $COMMENT"
fi

for file in ./"xtool - "*.scpt; do
  xattr -w com.apple.metadata:kMDItemComment "$COMMENT" "$file"
  xattr -w com.apple.metadata:kMDItemFinderComment "$COMMENT" "$file"
  xattr -w com.dzombak.xtool.installdate "$INSTALL_DATE" "$file"
  if [ -n "$XTOOL_VERSION" ]; then
    xattr -w com.dzombak.xtool.version "$XTOOL_VERSION" "$file"
  fi
  cp -vf "$file" "$HOME/Library/Scripts/Applications/Finder"
done
