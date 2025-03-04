#!/usr/bin/env bash
# This script is used to manually tile the currently active window into one of the tiling zones.
# Tiling zones are defined in the four arrays below. The number of the tiling zone is used to
# index the arrays.

# tile coordinates:
#            A B    C    D    E
pos_x_arr=(0 4 1645 1645 1645 2560)
pos_y_arr=(0 4 4    4    763  0)

# tile sizes:
#                A    B    C   D   E
size_x_arr=(2560 1637 913  913 913 1280)
size_y_arr=(1400 1392 1392 759 633 1024)

window_id=$(kwst get-active-window)

if [ $# -eq 0 ]; then
	echo "USAGE: kwin-tile TILING_ZONE"
	exit 1
fi

pos_x=${pos_x_arr[$1]}
# echo "pos_x: $pos_x"
pos_y=${pos_y_arr[$1]}
# echo "pos_y: $pos_y"
size_x=${size_x_arr[$1]}
# echo "size_x: $size_x"
size_y=${size_y_arr[$1]}
# echo "size_y: $size_y"

kwst set-window-geometry "$window_id" "$pos_x" "$pos_y" "$size_x" "$size_y"
