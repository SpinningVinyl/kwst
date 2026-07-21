# kwst -- KWin Scripting Tool

**kwst** is a small utility for controlling KWin-Wayland from the command line. 

It works by generating JS scripts on the fly, registering them with KWin, running them and receiving responses from the scripts over DBus. I know that it sounds a bit convoluted, but as far as I'm aware this is the only way to control KWin programmatically.

## Currently supported features

Here is the list of things that you can currently do with **kwst**:

- List all windows.
- Find a window.
- Get the UUID of the active window.
- Activate a window.
- Close a window.
- Get window geometry (size and position).
- Set window geometry (size and position).
- Set window properties (such as keepAbove, keepBelow, fullScreen, etc.)
- Get the number of the active workspace.
- Switch to a workspace.
- Send a window to a workspace.
- Get the absolute position of the mouse cursor.

## Wayland/X11

`kwst` is tested against KWin Wayland. While it *should* work with KWin X11, explicit support for X11 is deliberately out of scope. `kwst` functionality is easy to replicate under X11 with other tools, e.g. `wmctrl` and `xdotool`.

## Usage

Run `kwst --help` to get context-sensitive help. Run `kwst <command> --help` for more information on a command.

Also check shell scripts in the `_examples` directory to get an idea of what you can use **kwst** for.

## Quick start

List regular windows, including their captions:

```sh
kwst list --show-captions
```

Get the active window and inspect or change its geometry:

```sh
window_id=$(kwst get-active-window)
kwst get-window-geometry "$window_id"
kwst set-window-geometry "$window_id" 100 100 1200 800
```

Toggle the active window's always-on-top state:

```sh
window_id=$(kwst get-active-window)
kwst set-window-property --property=keepAbove --value=toggle "$window_id"
```

Find the first window whose caption begins with “Konsole” and activate it:

```sh
window_id=$(kwst find --search-field=caption '^Konsole' | head -n 1)
if [ -n "$window_id" ]; then
    kwst activate-window "$window_id"
fi
```

## Finding windows

The search term accepted by `find` is a case-insensitive JavaScript regular
expression, not a literal string. Quote it to prevent the shell from expanding
regular-expression characters:

```sh
kwst find --search-field=caption 'terminal|konsole'
kwst find --search-field=resourceClass '^org\.kde\.konsole$'
```

The searchable fields are `resourceClass` (the default), `resourceName`, and
`caption`. Invalid regular expressions are reported on standard error and
return exit status 1.

## Optional previous-window shortcut

The [`kwin-previous-window-script`](kwin-previous-window-script/README.md)
directory contains an optional resident KWin script that tracks window
activation and provides a shortcut for switching to the previously active
window. It is installed separately and is not required for any **kwst** command.

## Exit status

**kwst** exits with status 0 when a command succeeds and a non-zero status when setup, D-Bus, or KWin script execution fails. Commands that accept a window UUID return status 1 when no matching window exists. A timeout waiting for a KWin script to finish returns status 124.

## Running custom scripts

Since version 1.1.0, **kwst** supports running custom scripts. Please see the recommended script template in `custom-script-template.js` and example scripts in the `_examples` directory.

The custom scripts are parsed using Go's built-in `text/template` package. 

**kwst** supports passing up to six parameters to your custom scripts: 

```
kwst run-custom-script --parameter-1="value1" --parameter-2="value2" ... --parameter-6="value6" /path/to/script/file.js
```

Inside your custom scripts, `{{.P1}}` will be replaced with the value of parameter 1, `{{.P2}}` will be replaced with the value of parameter 2, etc.

When inserting a parameter as a JavaScript string, pass it through `jsString`:

```javascript
const value = {{jsString .P1}};
const regExp = new RegExp({{jsString .P2}}, "i");
```

`jsString` returns a complete quoted JavaScript string literal and escapes
quotes, backslashes, control characters, and line separators. Do not add quotes
around the template expression: use `{{jsString .P1}}`, not
`"{{jsString .P1}}"`.

You can also use comparisons:

```
{{if (eq .P1 "value1")}}do something{{else}}do something else{{end}}
```

You can find the full documentation of the Go text templating engine [here](https://pkg.go.dev/text/template).

## Installation

**kwst** is distributed as a single statically-linked binary. Just copy or symlink it to a directory that is listed in your `PATH` environment variable, e.g. `/usr/local/bin`. If you want to install the manpage, copy `kwst.1.gz` to `/usr/local/share/man/man1`.

To build from source, you would need GNU make and Go v1.23 or later. First clone the repository:

```
git clone https://github.com/SpinningVinyl/kwst.git && cd kwst
```

Then compile and install the program:

```
make && sudo make install
```

If you have [scdoc](https://git.sr.ht/~sircmpwn/scdoc) installed, this should also generate and install the man page.

## Integration tests

An opt-in integration suite exercises the compiled program against a live KWin
session using KDialog fixture windows. See [`integration/README.md`](integration/README.md)
for prerequisites, safety notes, and execution instructions.

## License

The project is licensed under the terms of GNU GPLv2 or later license.

## See also

- [KWin scripting API documentation](https://develop.kde.org/docs/plasma/kwin/api/)
