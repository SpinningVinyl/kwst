KWST(1)

# NAME

kwst -- KWin Scripting Tool

# SYNOPSIS

*kwst* <command> [_flags_]

# DESCRIPTION

The KWin Scripting Tool is designed to control some functions of the K Window
Manager (used by KDE Plasma) from the command line. It was written because no
such tools currently target kwin_wayland. The main purpose of the tool is to
provide something similar to _wmctrl_ and other tools used to control X11 window
managers implementing NET_WM and EWMH specifications.

# GLOBAL FLAGS

	*-h*, *--help*++
	Show context-sensitive help.
	
	*-d*, *--debug*++
	Enable debug mode.
	
	*--version*++
	Show version information and exit.

# CONTEXT-SENSITIVE HELP

Run "kwst <command> --help" for more information on a command.

# COMMANDS

	*list* [_flags_]
		List all windows. The data is returned as tab-separated rows containing the
		window's UUID, resourceClass and resourceName. Each window is represented
		by a separate row.

		The currently supported flags are:

		*-s*, *--include-special-windows*
			Include special windows that are not meant to be manipulated, e.g.
			plasmashell panels, desktop, etc. Such windows are not listed by
			default.
		*-c*, *--show-caption*
			Show window captions in the list.
		*-p*, *--show-pids*
			Show the PID of the process that the window belongs to (the PID is
			not guaranteed to be correct for X11 windows).

	*find* <search-term> [_flags_]
		Search for windows using the specified search term.

		The currently supported flags are:

		*-f*, *--search-field*="_resourceClass_"
			Specify the field to search in. Possible values: 'resourceClass'
			(default), 'resourceName', 'caption'.
			
	*get-active-window*
		Get the UUID of the active window.

	*get-window-geometry* <uuid>
		Get the geometry (size and position) of the window with the specified UUID.
		The data is returned in the format required for the set-window-geometry
		command (x y width height).

	*get-workspace*
		Get the ID of the active workspace.

	*set-workspace* <workspace-id>
		Switch to the workspace with the specified ID.

	*activate-window* <uuid>
		Activate the window with the provided UUID, if such a window exists.

	*previous-window*
		Switch to the second most recently active window. This command can be used
		to quickly switch between two windows.

	*set-window-size* <uuid> <width> <height>
		Set the size of the window with the provided UUID, provided that
		such a window exists.		

	*set-window-position* <uuid> <x> <y>
		Set the position of the window with the provided UUID, provided
		that such a window exists.

	*set-window-geometry* <uuid> <x> <y> <width> <height>
		Change geometry of the window with the provided UUID, provided
		that such a window exists.

	*set-window-workspace* <uuid> <workspace-id>
		Send the window with the specified UUID to the workspace with the specified
		number, provided that both exist.

	*set-window-property* _--property_=STRING _--value_=STRING <uuid>
		Change the value of a property on a window with the specified UUID.

		The required flags are:

		*-p*, *--property=STRING*
			Property to change the value of. Possible values of this flag:
			'keepAbove', 'keepBelow', 'shade', 'fullScreen', 'skipTaskbar',
			'skipPager', 'skipSwitcher'
		*-v*, *--value=STRING*
			Possible values: 'true', 'false', 'toggle'

	*close-window* <uuid>
		Close the window with the provided UUID, provided that it exists.

	*run-custom-script* [_parameters_] <script-file> 
		Run a custom script. Supports up to six optional parameters.

		Optional parameters:

		*--parameter-1=STRING*++
*--parameter-2=STRING*++
*--parameter-3=STRING*++
*--parameter-4=STRING*++
*--parameter-5=STRING*++
*--parameter-6=STRING*

# RUNNING CUSTOM SCRIPTS

Starting with version 1.0.0, *kwst* supports running custom scripts. Please see
the recommended script template supplied with the program for further details.

Custom scripts are parsed using Go's built-in 'text/template' package.

kwst supports passing up to six parameters to your custom scripts:

	*kwst* run-custom-script [[_--parameter-1="value1"_] ... [_--parameter-6="value6"_]]
	</path/to/script/file.js>

Inside your custom scripts, *{{.P1}}* will be replaced with the value of parameter 1,
*{{.P2}}* will be replaced with the value of parameter 2, etc.

You can also use comparisons:

	*{{if (eq .P1 "value1")}}*do something*{{else}}*do something else*{{end}}*

See Golang documentation for further information about Go's templating engine.

# AUTHORS

Written and maintained by Pavel Urusov. Please see _https://github.com/SpinningVinyl/kwst_
for up-to-date sources, etc.


