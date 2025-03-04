#!/usr/bin/env bash
# This script first checks if there is a window that matches the search criteria.
# If such a window is found, the script activates the window.
# If the window is not found, the script runs the program.

if [ $# -eq 0 ]; then
	echo "USAGE: activate PROGRAM_NAME"
fi

window_id=$(kwst find "$1" | head -n 1)
if [ "$window_id" = "" ]; then
	$1
else
	kwst activate-window "$window_id"
fi

