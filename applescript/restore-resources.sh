#!/usr/bin/env bash

set -uo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
pushd "$SCRIPT_DIR" || exit 1

find . -iname '*.scpt' -type f -print0 | while read -r -d $'\0' file
do
  if [ -f "$file.rsrc" ]; then
    echo "$file ..."
    Rez "$file.rsrc" -append -o "$file"
    SetFile -a C "$file"
  else
    echo "no .rsrc found for $file"
  fi
done

popd || exit
