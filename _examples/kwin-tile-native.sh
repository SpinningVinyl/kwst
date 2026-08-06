#!/usr/bin/env bash
# Assign the currently active window to a native KWin tile on the specified
# output.

if [ "$#" -ne 2 ]; then
	echo "USAGE: kwin-tile-native OUTPUT_ID TILE_PATH" >&2
	exit 2
fi

output_id=$1
tile_path=$2

window_id=$(kwst get-active-window) || exit
kwst set-window-tile --output="$output_id" "$window_id" "$tile_path"
