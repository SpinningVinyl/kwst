package main

type ListCmd struct {
	IncludeSpecialWindows bool `default:"false" short:"s" help:"Include special windows that are not meant to be manipulated, e.g. plasmashell panels, desktop, etc. Such windows are not listed by default."`
	ShowCaptions          bool `default:"false" short:"c" help:"Show window captions in the list."`
	ShowPids              bool `default:"false" short:"p" help:"Show the PID of the process that the window belongs to (the PID is not guaranteed to be correct for X11 windows)."`
}

type FindCmd struct {
	SearchField string `enum:"resourceClass,resourceName,caption" help:"Specify the field to search in. Possible values: resourceClass, resourceName, caption" short:"f" default:"resourceClass"`

	SearchTerm string `arg required help:"String to search for"`
}

type GetActiveWindowCmd struct{}

type GetWindowGeometryCmd struct {
	Uuid string `arg required help:"UUID of the window to manipulate"`
}

type GetWorkspaceCmd struct{}

type SetWorkspaceCmd struct {
	WorkspaceId int `arg required help:"Workspace number"`
}

type ActivateWindowCmd struct {
	Uuid string `arg required help:"UUID of the window you want to activate"`
}

type SetWindowSizeCmd struct {
	Uuid   string `arg required help:"UUID of the window to manipulate"`
	Width  int    `arg required`
	Height int    `arg required`
}

type SetWindowPosCmd struct {
	Uuid string `arg required help:"UUID of the window to manipulate"`
	X    int    `arg required`
	Y    int    `arg required`
}

type SetWindowGeometryCmd struct {
	Uuid   string `arg required help:"UUID of the window to manipulate"`
	X      int    `arg required`
	Y      int    `arg required`
	Width  int    `arg required`
	Height int    `arg required`
}

type SetWindowWorkspaceCmd struct {
	Uuid        string `arg required help:"UUID of the window to manipulate"`
	WorkspaceId int    `arg required help:"Number of the workspace to send the window to"`
}

type SetWindowPropertyCmd struct {
	Property string `required enum:"keepAbove,keepBelow,shade,fullScreen,skipTaskbar,skipPager,skipSwitcher" short:"p" help:"Property to change the value of. Possible values: keepAbove, keepBelow, shade, fullScreen, skipTaskbar, skipPager, skipSwitcher"`
	Value    string `required enum:"true,false,toggle" short:"v" help:"Possible values: true, false, toggle"`

	Uuid string `arg required help:"UUID of the window to manipulate"`
}

type CloseWindowCmd struct {
	Uuid string `arg required help:"UUID of the window to close"`
}
