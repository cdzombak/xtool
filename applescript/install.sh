#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

./restore-resources.sh
mkdir -p "$HOME/Library/Scripts/Applications/Finder"
cp -vf ./"xtool - "*.scpt "$HOME/Library/Scripts/Applications/Finder"
