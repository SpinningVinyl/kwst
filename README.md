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
- Get the number of the active workspace.
- Switch to a workspace.
- Send a window to a workspace.
- Set window properties (such as keepAbove, keepBelow, fullScreen, etc.)

## Wayland/X11

I personally use (and extensively test) **kwst** in Wayland, but as far as I understand, it should also be fully functional in KWin under X11 (since the scripting API remains the same). That said, I haven't tried it in X11 and presently I don't plan to fix any X11-only bugs, if such bugs exist.

## Usage

Run `kwst --help` to get context-sensitive help. Run `kwst <command> --help` for more information on a command.

Also check shell scripts in the `_examples` directory to get an idea of what you can use **kwst** for.

## Running custom scripts

Since version 1.1.0, **kwst** supports running custom scripts. Please see the recommended script template in `custom-script-template.js` and example scripts in the `_examples` directory.

The custom scripts are parsed using Go's built-in `text/template` package. 

**kwst** supports passing up to six parameters to your custom scripts: 

```
kwst run-custom-script --parameter-1="value1" --parameter-2="value2" ... --parameter-6="value6" </path/to/script/file.js>
```

Inside your custom scripts, `{{.P1}}` will be replaced with the value of parameter 1, `{{.P2}}` will be replaced with the value of parameter 2, etc.

You can also use comparisons:

```
{{if (eq .P1 "value1")}}do something{{else}}do something else{{end}}
```

You can find the full documentation of the Go text templating engine [here](https://pkg.go.dev/text/template).

## Installation

**kwst** is distributed as a single statically-linked binary. Just copy it to a directory that is listed in your `PATH` environment variable.

Or, if building from source (requires Go 1.23 or later), use `make && sudo make install`. 

## License

The project is licensed under the terms of GNU GPLv2 or later license.

## See also

- [KWin scripting API documentation](https://develop.kde.org/docs/plasma/kwin/api/)
