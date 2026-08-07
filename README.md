# kwst -- KWin Scripting Tool

**kwst** is a small utility for controlling KWin-Wayland from the command line.

It works by generating JS scripts on the fly, registering them with KWin, running them and receiving responses from the scripts over D-Bus. I know that it sounds a bit convoluted, but as far as I'm aware this is the only way to control KWin programmatically.

## Currently supported features

Here is the list of things that you can currently do with **kwst**:

- List all windows.
- Find a window.
- Get the UUID of the active window.
- Activate a window.
- Close a window.
- Get window geometry (size and position).
- Get window opacity.
- Change window opacity by setting it directly or using increase/decrease commands.
- Set window geometry (size and position).
- Set window geometry relative to the client area of its output.
- Set window properties (such as keepAbove, keepBelow, fullScreen, etc.)
- Work with KWin's native tiles: list available tiles, assign windows to tiles, or unassign windows from tiles.
- Get the number of the active workspace.
- Switch to a workspace.
- Send a window to a workspace.
- Get the absolute position of the mouse cursor.
- List enabled outputs.
- Get the active output as defined by KWin.
- Get the output containing the mouse cursor.
- Get the output containing the centre of the active window.
- Get output geometry, optionally excluding panels and other reserved areas.

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

If you have `column` installed, you can use it to format the output nicely:

```sh
kwst list | column -t -s "$(printf '\t')"
```

Get the active window and inspect or change its geometry or opacity:

```sh
window_id=$(kwst get-active-window)
kwst get-window-geometry "$window_id"
kwst set-window-geometry "$window_id" 100 100 1200 800
kwst set-window-geometry-relative "$window_id" 0 0 50 100
kwst get-window-opacity "$window_id"
kwst set-window-opacity "$window_id" 0.8
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

The searchable fields are `resourceClass` (used by default), `resourceName`, and
`caption`. Invalid regular expressions are reported on standard error and
return exit status 1.

## Optional previous-window shortcut

The [`kwin-previous-window-script`](kwin-previous-window-script/README.md)
directory contains an optional resident KWin script that tracks window
activation and provides a shortcut for switching to the previously active
window. It is installed separately and is not required for any **kwst** command.

## Exit status

**kwst** exits with status 0 when a command succeeds and a non-zero status when setup, D-Bus, or KWin script execution fails. Commands that accept a window UUID return status 1 when no matching window exists. A timeout waiting for a KWin script to finish returns status 124.

## Using KWin's native tiling

Since version 2.9.0, **kwst** supports KWin's native tiling. The following commands are available:

- `list-tiles [--output=OUTPUT] [--leaves-only]`. This command outputs the list of tiles for the specified output. If the output is not specified, the currently active output (as defined by KWin) is used instead. If the `--leaves-only` switch is used, the command will list only leaf tiles (i.e. tiles that do not contain other tiles).
- `get-window-tile <uuid>`. If the window with the specified UUID is assigned to a native tile, this command prints information about the tile.
- `set-window-tile [--output=OUTPUT] <uuid> <tile-path>`. This command assigns the window with the specified UUID to the tile with the specified locator path on the specified output. If no output is specified, the output containing the window is used instead.
- `unset-window-tile <uuid>`. If the window with the specified UUID is assigned to a tile, it will be unassigned from its tile.
- `list-tile-windows [--output=OUTPUT] [--show-captions] [--show-pids] <tile-path>`. This command outputs the list of all windows directly assigned to the specified tile.

### Locator paths

KWin does not expose any sort of persistent tile IDs. The tiling manager's tree is rebuilt every time a new user session starts. The locator paths remain unchanged across sessions only if the layout structure is unchanged.

For normal (non-floating) tiles the following rules apply as long as the layout remains unchanged:

- for horizontal layout tiles, children are enumerated left to right;
- for vertical layout tiles, children are enumerated top to bottom.

Let's assume we have the following layout: one tile on the left that spans the whole height of the screen, and another tile on the right that is split into two tiles vertically. In that case, tile paths would be as follows:

- root: `.`
- left: `0`
- right: `1`
- top right: `1.0`
- bottom right: `1.1`

It should also be noted that KWin maintains one tile tree per output and (since KWin 6.4) per virtual desktop. This means that if you use different tile layouts on different virtual desktops, locator paths of individual tiles will be different between them as well.

Internally, KWin tracks tile associations of individual windows using the `window.tile` property. If a window belongs to a single virtual desktop, this property will not change when the active virtual desktop changes. However, if a window belongs to multiple virtual desktops, `window.tile` will change according to the currently active desktop -- the window can be assigned to different tiles on different desktops, or assigned on some but not the others.

Hypothetically, let's say a window is present on desktop 1 and desktop 2; it is assigned to tile 1.0 on desktop 1 and not assigned to any tile on desktop 2. In that case, `window.tile` would point to tile 1.0 when desktop 1 is active, but it would be `null` when desktop 2 is active. This means that calling `get-window-tile` with the window's UUID when desktop 2 is active would return a message that the window is not tiled, despite the fact that it's tiled on desktop 1.

Now let's say we switch to desktop 3, on which the window in question is not present. Since the window is not present on desktop 3, the value of `window.tile` would not be updated by KWin. This means that this property can either be `null` or point to the tile 1.0 on desktop 1. But even in the latter case, there is no way for **kwst** to determine the desktop which the tile belongs to. In cases like that, the `get-window-tile` command returns an error message saying that it is impossible to determine the virtual desktop for the window's tile.

### Tile layouts vs quick tiles

Confusingly, KWin exposes more than one kind of tiling. There are persistent user-defined tile layouts that can be edited in the tile manager, and there are so-called "quick tile layouts" that allow users to quickly tile windows to the left, right, top, bottom, top-left, top-right, bottom-left, and bottom-right. The only **kwst** command that supports quick tiles is `unset-window-tile`, which can remove any kind of tile association. All other commands do not support quick tiles.

### Suggested workflow

Suggested workflow for working with KWin's native tiles:

First, open the tile manager (by default it's tied to the `Logo+T` keyboard shortcut) and create a tile layout. Then run the following command to enumerate the tiles:

```sh
kwst list-tiles | column -t -s "$(printf '\t')"
```

If you have more than one monitor, you first need to enumerate outputs:

```sh
kwst list-outputs
```

After that you can list tiles for each individual output:

```sh
kwst list-tiles --output=DP-1 | column -t -s "$(printf '\t')"
```

Take note of the tile locator paths. You can now use them in your scripts; see [`kwin-tile-native.sh`](_examples/kwin-tile-native.sh) for an example.

## Running custom scripts

Since version 1.1.0, **kwst** supports running custom scripts. Please see the recommended script template in `custom-script-template.js` and example scripts in the `_examples` directory.

The custom scripts are parsed using Go's built-in `text/template` package.

**kwst** supports passing up to six parameters to your custom scripts:

```sh
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

Please see the [Go text/template documentation](https://pkg.go.dev/text/template) for more information.

## Debugging KWin scripts

The `kwst-debug-listener` executable implements the same `Complete` D-Bus
callback as **kwst**, but prints every call it receives instead of processing
the result or exiting. This makes it possible to
debug scripts interactively with KWin's scripting console.

Build and start the listener:

```sh
make
./build/kwst-debug-listener
```

The listener prints its D-Bus address when it starts. Open the KWin scripting
console with:

```sh
plasma-interactiveconsole --kwin
```

Use the displayed address as `dbusAddr` in the script being tested, then run the
script in the console. Calls to the listener's `Complete` D-Bus method are
printed to its standard output with timestamps without stopping the listener,
so the script can be run repeatedly. Press `q` in the listener's terminal to
quit.

## Installation

**kwst** itself is a single statically linked binary. Just copy or symlink it to a directory that is listed in your `PATH` environment variable, e.g. `/usr/local/bin`. If you want to install the man page, copy `kwst.1.gz` to `/usr/local/share/man/man1`.

Starting with version 3.0.0, the project also includes a debug listener (`kwst-debug-listener`), which is useful for developing custom scripts using KWin's scripting console.

To build from source, you need GNU make and Go v1.23 or later. First clone the repository:

```sh
git clone https://github.com/SpinningVinyl/kwst.git && cd kwst
```

Then compile and install the program:

```sh
make && sudo make install
```

If you have [scdoc](https://git.sr.ht/~sircmpwn/scdoc) installed, this should also generate and install the man page.

## Integration tests

An opt-in integration suite exercises the compiled program against a live KWin
session using KDialog fixture windows. See [`integration/README.md`](integration/README.md)
for prerequisites, safety notes, and execution instructions.

## AI use disclosure

Since version 1.4.0, **kwst** has been developed with AI assistance. AI is primarily used for targeted edits and test development under maintainer review.

The KWin-facing JavaScript and the majority of the Go client are written and maintained manually. JavaScript changes are tested using KWin’s scripting console and the live integration suite.

The live integration suite was designed and initially implemented manually. AI assistance is used for many additional integration and unit test cases.

AI is also used for code reviews and figuring out edge cases.

## License

The project is licensed under the GNU General Public License, version 2 or later.

## See also

- [KWin scripting API documentation](https://develop.kde.org/docs/plasma/kwin/api/)
