# kwst -- KWin Scripting Tool

**kwst** is a small utility for controlling KWin-Wayland from the command line. 

It works by generating JS scripts on the fly, registering them with KWin, running them and receiving responses from the scripts over DBus. I know that it sounds a bit convoluted, but as far as I'm aware this is the only way to control KWin programmatically.

## Currently supported features

Here is the list of things that you can currently do with **kwst**:

- List all windows.
- Find a window.
- Get the UUID of the active window.
- Activate a window.
- Get window geometry (size and position).
- Set window geometry (size and position).
- Get the number of the active workspace.
- Switch to a workspace.
- Send a window to a workspace.

## Usage

Run `kwst --help` to get context-sensitive help. Run `kwst <command> --help` for more information on a command.

Also check shell scripts in the `_examples` directory to get an idea of what you can use **kwst** for.

## Installation

**kwst** is distributed as a single statically-linked binary. Just copy it to a directory that is listed in your `PATH` environment variable.

## License

The project is licensed under the terms of GNU GPLv2 or later license.

## See also

- [KWin scripting API documentation](https://develop.kde.org/docs/plasma/kwin/api/)
