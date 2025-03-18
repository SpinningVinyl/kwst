package main

import (
	"os"
)

type ListCmd struct {
	IncludeSpecialWindows bool `default:"false" short:"s" help:"Include special windows that are not meant to be manipulated, e.g. plasmashell panels, desktop, etc. Such windows are not listed by default."`
	ShowCaptions          bool `default:"false" short:"c" help:"Show window captions in the list."`
	ShowPids              bool `default:"false" short:"p" help:"Show the PID of the process that the window belongs to (the PID is not guaranteed to be correct for X11 windows)."`
}

func (lc *ListCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_LIST
	sp.Params.IncludeSpecialWindows = lc.IncludeSpecialWindows
	sp.Params.ShowCaptions = lc.ShowCaptions
	sp.Params.ShowPids = lc.ShowPids
	return nil
}

type FindCmd struct {
	SearchField string `enum:"resourceClass,resourceName,caption" help:"Specify the field to search in. Possible values: resourceClass, resourceName, caption" short:"f" default:"resourceClass"`

	SearchTerm string `arg:"" required:"" help:"String to search for"`
}

func (fc *FindCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_FIND
	sp.Params.SearchTerm = fc.SearchTerm
	sp.Params.SearchField = fc.SearchField
	return nil
}

type GetActiveWindowCmd struct{}

func (gawc *GetActiveWindowCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_ACTIVE_WINDOW
	return nil
}

type GetWindowGeometryCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to manipulate"`
}

func (gwgc *GetWindowGeometryCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_WINDOW_GEOMETRY
	sp.Params.Uuid = gwgc.Uuid
	return nil
}

type GetWorkspaceCmd struct{}

func (gwc *GetWorkspaceCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_WORKSPACE
	return nil
}

type SetWorkspaceCmd struct {
	WorkspaceId int `arg:"" required:"" help:"Workspace number"`
}

func (swc *SetWorkspaceCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WORKSPACE
	sp.Params.WorkspaceId = swc.WorkspaceId
	return nil
}

type ActivateWindowCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window you want to activate"`
}

func (awc *ActivateWindowCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_ACTIVATE_WINDOW
	sp.Params.Uuid = awc.Uuid
	return nil
}

type SetWindowSizeCmd struct {
	Uuid   string `arg:"" required:"" help:"UUID of the window to manipulate"`
	Width  int    `arg:"" required:""`
	Height int    `arg:"" required:""`
}

func (swsc *SetWindowSizeCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WINDOW_SIZE
	sp.Params.Uuid = swsc.Uuid
	sp.Params.Width = swsc.Width
	sp.Params.Height = swsc.Height
	return nil
}

type SetWindowPosCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to manipulate"`
	X    int    `arg:"" required:""`
	Y    int    `arg:"" required:""`
}

func (swpc *SetWindowPosCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WINDOW_POSITION
	sp.Params.Uuid = swpc.Uuid
	sp.Params.X = swpc.X
	sp.Params.Y = swpc.Y
	return nil
}

type SetWindowGeometryCmd struct {
	Uuid   string `arg:"" required:"" help:"UUID of the window to manipulate"`
	X      int    `arg:"" required:""`
	Y      int    `arg:"" required:""`
	Width  int    `arg:"" required:""`
	Height int    `arg:"" required:""`
}

func (swgc *SetWindowGeometryCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WINDOW_GEOMETRY
	sp.Params.Uuid = swgc.Uuid
	sp.Params.X = swgc.X
	sp.Params.Y = swgc.Y
	sp.Params.Width = swgc.Width
	sp.Params.Height = swgc.Height
	return nil
}

type SetWindowWorkspaceCmd struct {
	Uuid        string `arg:"" required:"" help:"UUID of the window to manipulate"`
	WorkspaceId int    `arg:"" required:"" help:"Number of the workspace to send the window to"`
}

func (swwc *SetWindowWorkspaceCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WINDOW_WORKSPACE
	sp.Params.Uuid = swwc.Uuid
	sp.Params.WorkspaceId = swwc.WorkspaceId
	return nil
}

type SetWindowPropertyCmd struct {
	Property string `required:"" enum:"keepAbove,keepBelow,shade,fullScreen,skipTaskbar,skipPager,skipSwitcher" short:"p" help:"Property to change the value of. Possible values: keepAbove, keepBelow, shade, fullScreen, skipTaskbar, skipPager, skipSwitcher"`
	Value    string `required:"" enum:"true,false,toggle" short:"v" help:"Possible values: true, false, toggle"`

	Uuid string `arg:"" required:"" help:"UUID of the window to manipulate"`
}

func (swpc *SetWindowPropertyCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WINDOW_PROPERTY
	sp.Params.Uuid = swpc.Uuid
	sp.Params.WindowProperty = swpc.Property
	sp.Params.PropertyValue = swpc.Value
	return nil
}

type CloseWindowCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to close"`
}

func (cwc *CloseWindowCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_CLOSE_WINDOW
	sp.Params.Uuid = cwc.Uuid
	return nil
}

type RunCustomScriptCmd struct {
	Parameter1 string
	Parameter2 string
	Parameter3 string
	Parameter4 string
	Parameter5 string
	Parameter6 string

	ScriptFile *os.File `arg:"" required:"" help:"Path to the script template."`
}

func (rcsc *RunCustomScriptCmd) Run(sp *ScriptPackage) error {
	bytes, err := os.ReadFile(rcsc.ScriptFile.Name())
	if err != nil {
		return err
	}
	defer rcsc.ScriptFile.Close()
	sp.ScriptTemplate = string(bytes)

	sp.Params.P1 = rcsc.Parameter1
	sp.Params.P2 = rcsc.Parameter2
	sp.Params.P3 = rcsc.Parameter3
	sp.Params.P4 = rcsc.Parameter4
	sp.Params.P5 = rcsc.Parameter5
	sp.Params.P6 = rcsc.Parameter6

	return nil
}
