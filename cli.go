package main

// define the CLI structure for Kong to parse. See also commands.go
type Globals struct {
	Debug bool `help:"Enable debug mode." short:"d"`
}

type CLI struct {
	Globals

	List                      ListCmd                      `cmd:"" help:"List all windows. The data is returned as tab-separated rows containing the window's UUID, resourceClass and resourceName. Each window is represented by a separate row."`
	Find                      FindCmd                      `cmd:"" help:"Search for windows using a case-insensitive regular expression."`
	GetActiveWindow           GetActiveWindowCmd           `cmd:"" help:"Get the UUID of the active window."`
	GetWindowGeometry         GetWindowGeometryCmd         `cmd:"" help:"Get the geometry (size and position) of the window with the specified UUID. The data is returned in the format required for the set-window-geometry command (x y width height)."`
	GetWorkspace              GetWorkspaceCmd              `cmd:"" help:"Get the ID of the active workspace."`
	SetWorkspace              SetWorkspaceCmd              `cmd:"" help:"Switch to the workspace with the specified ID."`
	ActivateWindow            ActivateWindowCmd            `cmd:"" help:"Activate the window with the provided UUID, if such a window exists."`
	SetWindowSize             SetWindowSizeCmd             `cmd:"" help:"Set the size of the window with the provided UUID."`
	SetWindowPosition         SetWindowPosCmd              `cmd:"" help:"Set the position of the window with the provided UUID."`
	SetWindowGeometry         SetWindowGeometryCmd         `cmd:"" help:"Change geometry of the window with the provided UUID."`
	SetWindowWorkspace        SetWindowWorkspaceCmd        `cmd:"" help:"Send the window with the specified UUID to the workspace with the specified number."`
	SetWindowProperty         SetWindowPropertyCmd         `cmd:"" help:"Change the value of a property on a window with the specified UUID."`
	CloseWindow               CloseWindowCmd               `cmd:"" help:"Close the window with the provided UUID."`
	RunCustomScript           RunCustomScriptCmd           `cmd:"" help:"Run a custom script. Supports up to six optional parameters."`
	GetMousePosition          MousePosCmd                  `cmd:"" help:"Return the absolute position of the mouse cursor."`
	ListOutputs               ListOutputsCmd               `cmd:"" help:"Return the list of enabled outputs."`
	GetActiveOutput           GetActiveOutputCmd           `cmd:"" help:"Return the active output (as defined by KWin)."`
	GetCursorOutput           GetCursorOutputCmd           `cmd:"" help:"Return the output containing the mouse cursor."`
	GetActiveWindowOutput     GetActiveWindowOutputCmd     `cmd:"" help:"Return the output containing the geometric centre of the active window."`
	GetOutputGeometry         GetOutputGeometryCmd         `cmd:"" help:"Return the geometry (size and position) of the specified output. The data is returned in the following format: x y width height."`
	GetWindowOpacity          GetWindowOpacityCmd          `cmd:"" help:"Return the opacity of the specified window (from 0.0 to 1.0)."`
	SetWindowOpacity          SetWindowOpacityCmd          `cmd:"" help:"Set the opacity of the window with the provided UUID, if such a window exists."`
	IncreaseWindowOpacity     IncreaseWindowOpacityCmd     `cmd:"" help:"Increase the opacity of the window with the provided UUID by 0.05. Opacity can't be set higher than 1.0."`
	DecreaseWindowOpacity     DecreaseWindowOpacityCmd     `cmd:"" help:"Decrease the opacity of the window with the provided UUID by 0.05. Opacity can't be set lower than 0.1."`
	SetWindowGeometryRelative SetWindowGeometryRelativeCmd `cmd:"" help:"Change the geometry of the window with the provided UUID relative to the screen's client area dimensions (in percent)."`
	SetWindowSizeRelative     SetWindowSizeRelativeCmd     `cmd:"" help:"Change the size of the window with the provided UUID relative to the screen's client area dimensions (in percent)."`
	SetWindowPositionRelative SetWindowPositionRelativeCmd `cmd:"" help:"Change the position of the window with the provided UUID relative to the screen's client area dimensions (in percent)."`
	ListTiles                 ListTilesCmd                 `cmd:"" help:"List tiles configured in KWin's tile manager."`
	GetWindowTile             GetWindowTileCmd             `cmd:"" help:"Get the tile that the window with the provided UUID belongs to, provided such a window and such a tile exist."`
	SetWindowTile             SetWindowTileCmd             `cmd:"" help:"Assign the window with the provided UUID to the tile with the provided locator path (optionally on the specified output)."`
	UnsetWindowTile           UnsetWindowTileCmd           `cmd:"" help:"Unassign the window with the provided UUID from its current tile, if such a tile exists."`
	ListTileWindows           ListTileWindowsCmd           `cmd:"" help:"List all windows assigned to the tile with the provided locator path."`
	ResizeTile                ResizeTileCmd                `cmd:"" help:"Resize the tile with the provided locator path (optionally on the specified output)."`
	ResizeActiveTile          ResizeActiveTileCmd          `cmd:"" help:"Resize the tile that the active window is assigned to."`
	SetTileGeometry           SetTileGeometryCmd           `cmd:"" help:"Sets the relative geometry of the tile with the provided locator path (optionally on the specified output)."`
	SetActiveTileGeometry     SetActiveTileGeometryCmd     `cmd:"" help:"Sets the relative geometry of the tile that the active window is assigned to, provided that such a tile exists."`
}
