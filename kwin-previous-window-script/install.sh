#!/bin/sh

# SPDX-FileCopyrightText: 2026 Pavel Urusov
# SPDX-License-Identifier: GPL-2.0-or-later

set -eu

package_id="net.prsv.kwst.previouswindow"
script_dir=$(dirname -- "$0")
script_dir=$(CDPATH='' cd -- "$script_dir" && pwd)

if ! kpackage_tool=$(command -v kpackagetool6); then
    echo "Error: kpackagetool6 is not installed or is not available in PATH." >&2
    exit 1
fi

if "$kpackage_tool" --type=KWin/Script --show "$package_id" >/dev/null 2>&1; then
    "$kpackage_tool" --type=KWin/Script --remove "$package_id"
fi

"$kpackage_tool" --type=KWin/Script --install "$script_dir"
