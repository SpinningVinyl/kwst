package main

type ListCmd struct {
    IncludeSpecialWindows bool `default:"false" short:"s" help:"Include special windows that are not meant to be manipulated, e.g. plasmashell panels, desktop, etc. Such windows are not listed by default."`
}

type FindCmd struct {
    SearchField string `enum:"resourceClass,resourceName,caption" help:"Specify the field to search in. Possible values: resourceClass, resourceName, caption" short:"f" default:"resourceClass"`
    
    SearchTerm string `arg required help:"String to search for"`
}

type GetActiveWindowCmd struct {}

type GetWindowGeometryCmd struct {
    Uuid string `arg required help:"UUID of the window to manipulate"`    
}

type GetWorkspaceCmd struct {}

type SetWorkspaceCmd struct {
    Id int `arg required help:"Workspace number"`
}

type ActivateWindowCmd struct {
    Uuid string `arg required help:"UUID of the window you want to activate"`
}

type SetWindowSizeCmd struct {
    Uuid string `arg required help:"UUID of the window to manipulate"`
    Width int `arg required`
    Height int `arg required`
}

type SetWindowPosCmd struct {
    Uuid string `arg required help:"UUID of the window to manipulate"`    
    X int `arg required`
    Y int `arg required`
}

type SetWindowWorkspaceCmd struct {
    Uuid string `arg required help:"UUID of the window to manipulate"`
    WorkspaceId int `arg required help:"Number of the workspace to send the window to"`
}
