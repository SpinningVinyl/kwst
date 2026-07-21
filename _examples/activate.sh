#!/usr/bin/env bash
# This script first checks if there is a window that matches the regular
# expression. If such a window is found, the script activates it. Otherwise,
# the script runs the provided program.

if [ "$#" -lt 2 ]; then
	echo "USAGE: activate SEARCH_REGEX PROGRAM [ARGUMENT ...]" >&2
	exit 2
fi

search_regex=$1
shift

matches=$(kwst find --search-field=resourceClass "$search_regex") || exit
window_id=${matches%%$'\n'*}
if [ -z "$window_id" ]; then
	"$@"
else
	kwst activate-window "$window_id"
fi
