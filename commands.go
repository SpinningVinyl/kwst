package main

import (
	"fmt"
	"math"
	"os"
)

const (
	minRelativeSizePercent = 10
	maxRelativePercent     = 100
)

type ListCmd struct {
	IncludeSpecialWindows bool `default:"false" short:"s" help:"Include special windows that are not meant to be manipulated, e.g. plasmashell panels, desktop, etc. Such windows are not listed by default."`
	ShowCaptions          bool `default:"false" short:"c" help:"Show window captions in the list."`
	ShowPids              bool `default:"false" short:"p" help:"Show the PID of the process that the window belongs to (the PID is not guaranteed to be correct for X11 windows)."`
}

func (lc *ListCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_WINDOW_LIST_HELPERS + JS_LIST
	sp.Params.IncludeSpecialWindows = lc.IncludeSpecialWindows
	sp.Params.ShowCaptions = lc.ShowCaptions
	sp.Params.ShowPids = lc.ShowPids
	return nil
}

type FindCmd struct {
	SearchField string `enum:"resourceClass,resourceName,caption" help:"Specify the field to search in. Possible values: resourceClass, resourceName, caption" short:"f" default:"resourceClass"`

	SearchTerm string `arg:"" required:"" help:"Case-insensitive JavaScript regular expression to search for"`
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
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_SIZE
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
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_POSITION
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
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_GEOMETRY
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
	sp.Custom = true
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

type MousePosCmd struct{}

func (mpc *MousePosCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_MOUSE_POS
	return nil
}

type ListOutputsCmd struct{}

func (loc ListOutputsCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_LIST_OUTPUTS
	return nil
}

type GetActiveOutputCmd struct{}

func (gaoc GetActiveOutputCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_ACTIVE_OUTPUT
	return nil
}

type GetCursorOutputCmd struct{}

func (gcoc GetCursorOutputCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_CURSOR_OUTPUT
	return nil
}

type GetActiveWindowOutputCmd struct{}

func (gawoc GetActiveWindowOutputCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_ACTIVE_WINDOW_OUTPUT
	return nil
}

type GetOutputGeometryCmd struct {
	ClientArea bool   `default:"false" short:"c" help:"Return the maximum client area which excludes panels and other surfaces that reserve space through KWin's strut mechanism."`
	OutputName string `arg:"" required:"" help:"Name of the output to return the geometry of."`
}

func (gogc GetOutputGeometryCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_OUTPUT_GEOMETRY
	sp.Params.ClientArea = gogc.ClientArea
	sp.Params.OutputName = gogc.OutputName
	return nil
}

type GetWindowOpacityCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to get the opacity of."`
}

func (gwoc GetWindowOpacityCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_GET_WINDOW_OPACITY
	sp.Params.Uuid = gwoc.Uuid
	return nil
}

type SetWindowOpacityCmd struct {
	Uuid    string  `arg:"" required:"" help:"UUID of the window to change the opacity of."`
	Opacity float64 `arg:"" required:"" help:"Opacity from 0.1 to 1.0 (inclusive)."`
}

func (swoc SetWindowOpacityCmd) Validate() error {
	if math.IsNaN(swoc.Opacity) ||
		math.IsInf(swoc.Opacity, 0) ||
		swoc.Opacity < 0.1 ||
		swoc.Opacity > 1.0 {
		return fmt.Errorf("opacity must be between 0.1 and 1.0")
	}
	return nil
}

func (swoc SetWindowOpacityCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_SET_WINDOW_OPACITY
	sp.Params.Opacity = swoc.Opacity
	sp.Params.Uuid = swoc.Uuid
	return nil
}

type IncreaseWindowOpacityCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to increase the opacity of."`
}

func (iwoc IncreaseWindowOpacityCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_INCREASE_WINDOW_OPACITY
	sp.Params.Uuid = iwoc.Uuid
	return nil
}

type DecreaseWindowOpacityCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to decrease the opacity of."`
}

func (dwoc DecreaseWindowOpacityCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_DECREASE_WINDOW_OPACITY
	sp.Params.Uuid = dwoc.Uuid
	return nil
}

type SetWindowGeometryRelativeCmd struct {
	Uuid           string  `arg:"" required:"" help:"UUID of the window to change the geometry of."`
	RelativeX      float64 `arg:"" required:""`
	RelativeY      float64 `arg:"" required:""`
	RelativeWidth  float64 `arg:"" required:""`
	RelativeHeight float64 `arg:"" required:""`
}

func (cmd *SetWindowGeometryRelativeCmd) Validate() error {
	if math.IsNaN(cmd.RelativeX) ||
		math.IsNaN(cmd.RelativeY) ||
		math.IsNaN(cmd.RelativeWidth) ||
		math.IsNaN(cmd.RelativeHeight) ||
		math.IsInf(cmd.RelativeX, 0) ||
		math.IsInf(cmd.RelativeY, 0) ||
		math.IsInf(cmd.RelativeWidth, 0) ||
		math.IsInf(cmd.RelativeHeight, 0) {
		return fmt.Errorf("relative X, Y, width and height must be valid numbers")
	}
	cmd.RelativeX = min(max(cmd.RelativeX, 0), maxRelativePercent)
	cmd.RelativeY = min(max(cmd.RelativeY, 0), maxRelativePercent)
	cmd.RelativeWidth = min(max(cmd.RelativeWidth, minRelativeSizePercent), maxRelativePercent)
	cmd.RelativeHeight = min(max(cmd.RelativeHeight, minRelativeSizePercent), maxRelativePercent)
	return nil
}

func (cmd SetWindowGeometryRelativeCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_GEOMETRY_RELATIVE
	sp.Params.Uuid = cmd.Uuid
	sp.Params.RelativeX = cmd.RelativeX
	sp.Params.RelativeY = cmd.RelativeY
	sp.Params.RelativeWidth = cmd.RelativeWidth
	sp.Params.RelativeHeight = cmd.RelativeHeight
	return nil
}

type SetWindowSizeRelativeCmd struct {
	Uuid           string  `arg:"" required:"" help:"UUID of the window to change the size of."`
	RelativeWidth  float64 `arg:"" required:""`
	RelativeHeight float64 `arg:"" required:""`
}

func (cmd *SetWindowSizeRelativeCmd) Validate() error {
	if math.IsNaN(cmd.RelativeWidth) ||
		math.IsNaN(cmd.RelativeHeight) ||
		math.IsInf(cmd.RelativeWidth, 0) ||
		math.IsInf(cmd.RelativeHeight, 0) {
		return fmt.Errorf("relative width and height must be valid numbers")
	}
	cmd.RelativeWidth = min(max(cmd.RelativeWidth, minRelativeSizePercent), maxRelativePercent)
	cmd.RelativeHeight = min(max(cmd.RelativeHeight, minRelativeSizePercent), maxRelativePercent)
	return nil
}

func (cmd SetWindowSizeRelativeCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_SIZE_RELATIVE
	sp.Params.Uuid = cmd.Uuid
	sp.Params.RelativeWidth = cmd.RelativeWidth
	sp.Params.RelativeHeight = cmd.RelativeHeight
	return nil
}

type SetWindowPositionRelativeCmd struct {
	Uuid      string  `arg:"" required:"" help:"UUID of the window to change the geometry of."`
	RelativeX float64 `arg:"" required:""`
	RelativeY float64 `arg:"" required:""`
}

func (cmd *SetWindowPositionRelativeCmd) Validate() error {
	if math.IsNaN(cmd.RelativeX) ||
		math.IsNaN(cmd.RelativeY) ||
		math.IsInf(cmd.RelativeX, 0) ||
		math.IsInf(cmd.RelativeY, 0) {
		return fmt.Errorf("relative X and Y must be valid numbers")
	}
	cmd.RelativeX = min(max(cmd.RelativeX, 0), maxRelativePercent)
	cmd.RelativeY = min(max(cmd.RelativeY, 0), maxRelativePercent)
	return nil
}

func (cmd SetWindowPositionRelativeCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_SET_WINDOW_POSITION_RELATIVE
	sp.Params.Uuid = cmd.Uuid
	sp.Params.RelativeX = cmd.RelativeX
	sp.Params.RelativeY = cmd.RelativeY
	return nil
}

type ListTilesCmd struct {
	OutputName string `name:"output" help:"Name of the output containing the root tile."`
	LeavesOnly bool   `default:"false" help:"List only leaf tiles, omitting layout tiles."`
}

func (cmd ListTilesCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_HELPERS + JS_LIST_TILES
	sp.Params.OutputName = cmd.OutputName
	sp.Params.LeavesOnly = cmd.LeavesOnly
	return nil
}

type GetWindowTileCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to get the tile association of."`
}

func (cmd GetWindowTileCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_HELPERS + JS_GET_WINDOW_TILE
	sp.Params.Uuid = cmd.Uuid
	return nil
}

type SetWindowTileCmd struct {
	OutputName string `name:"output" help:"Name of the output containing the root tile."`
	Uuid       string `arg:"" required:"" help:"UUID of the window to be assigned to the specified tile."`
	TilePath   string `arg:"" required:"" help:"Locator path of the tile to assign the window to."`
}

func (cmd SetWindowTileCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_HELPERS + JS_SET_WINDOW_TILE
	sp.Params.Uuid = cmd.Uuid
	sp.Params.OutputName = cmd.OutputName
	sp.Params.TilePath = cmd.TilePath
	return nil
}

type UnsetWindowTileCmd struct {
	Uuid string `arg:"" required:"" help:"UUID of the window to unassign from its current tile."`
}

func (cmd UnsetWindowTileCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_UNASSIGNMENT_HELPER + JS_UNSET_WINDOW_TILE
	sp.Params.Uuid = cmd.Uuid
	return nil
}

type ListTileWindowsCmd struct {
	OutputName   string `name:"output" help:"Name of the output containing the root tile."`
	ShowCaptions bool   `default:"false" short:"c" help:"Show window captions in the list."`
	ShowPids     bool   `default:"false" short:"p" help:"Show the PID of the process that the window belongs to (the PID is not guaranteed to be correct for X11 windows)."`
	TilePath     string `arg:"" required:"" help:"Locator path of the tile."`
}

func (cmd ListTileWindowsCmd) Run(sp *ScriptPackage) error {
	sp.ScriptTemplate += JS_TILE_HELPERS + JS_WINDOW_LIST_HELPERS + JS_LIST_TILE_WINDOWS
	sp.Params.OutputName = cmd.OutputName
	sp.Params.ShowCaptions = cmd.ShowCaptions
	sp.Params.ShowPids = cmd.ShowPids
	sp.Params.TilePath = cmd.TilePath
	return nil
}
