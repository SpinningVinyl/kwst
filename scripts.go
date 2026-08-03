package main

var JS_HEADER string = `const dbusAddr = "{{.DbusAddr}}";

const debug = {{.Debug}};

const scriptName = "{{.ScriptName}}";

let exitCode = 0;

const debugLog = (msg) => {
    if (debug) {
        print(msg.toString());
    }
}

const close = () => {
    debugLog("Calling CloseWithStatus() on " + dbusAddr);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "CloseWithStatus", exitCode);
}

const returnResult = (msgBody) => {
    debugLog("RESULT: " + msgBody);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Msg", "result", msgBody.toString());
}

const returnError = (msgBody) => {
    exitCode = 1;
    debugLog("ERROR: " + msgBody);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Msg", "error", msgBody.toString());
}

const findWindow = (uuid) => {
    const window = workspace.windowList().find(
        candidate => candidate.internalId == uuid
    );

    return window;
}

const updateWindowGeometry = (window, changes) => {
    const geometry = Object.assign(
        {},
        window.frameGeometry,
        changes
    );

    window.frameGeometry = geometry;
    return geometry;
}

const clientAreaForWindow = (window) => {
    const output = window.output;
    const desktop = workspace.currentDesktopForScreen(output);
    return workspace.clientArea(KWin.MaximizeArea, output, desktop);
}

const findOutput = (outputName) => {
    const output = workspace.screens.find(
        (candidate) => candidate.name === outputName
    );
    return output;
}

const formatFloat = (num) => {
    const rounded = Number(num.toFixed(6));
    return Object.is(rounded, -0) ? "0" : String(rounded);
};

const enumerateTiles = (rootTile, leavesOnly, callback) => {
    function visit(tile, path) {
        const children = tile.tiles;
        const isLeaf = !tile.isLayout;

        if (!leavesOnly || isLeaf) {
            callback(tile, path, isLeaf);
        }

        for (let i = 0; i < children.length; i++) {
            const childPath = path === "."
            ? String(i)
            : path + "." + i;

            visit(children[i], childPath);
        }
    }

    visit(rootTile, ".");
}

const resolveTile = (root, path) => {
    if (path === ".") {
        return root;
    }
    let tile = root;
    for (const component of path.split(".")) {
        if (!/^(0|[1-9]\d*)$/.test(component)) {
            return null;
        }
        const index = Number(component);
        const children = tile.tiles;

        if (index >= children.length) {
            return null;
        }
        tile = children[index];
    }
    return tile;
}

const pathForTile = (tile) => {
    if (tile === null) {
        return null;
    }

    const components = [];
    let current = tile;

    while (current.parent !== null) {
        const index = current.positionInLayout;

        if (!Number.isInteger(index) || index < 0) {
            return null;
        }

        components.push(String(index));
        current = current.parent;
    }

    components.reverse();

    return {
        path: components.length === 0 ? "." : components.join("."),
        root: current,
    };
}

const assignToTile = (targetWindow, tile) => {
    if (typeof tile.manage === "function") {
        return tile.manage(targetWindow);
    } else {
        targetWindow.tile = tile;
        return targetWindow.tile === tile;
    }
}

const unassignFromTile = (targetWindow) => {
    const tile = targetWindow.tile;

    if (!tile) {
        return true;
    }

    if (typeof tile.unmanage === "function") {
        return tile.unmanage(targetWindow);
    }

    targetWindow.tile = null;
    return targetWindow.tile === null;
}

const isOnCurrentDesktop = (targetWindow, targetOutput) => {
    const currentDesktop = workspace.currentDesktopForScreen(targetOutput);
    const windowDesktops = targetWindow.desktops;

    return (
        windowDesktops.length === 0 || // window is on all desktops
        windowDesktops.includes(currentDesktop)
    );
}

debugLog(scriptName + " START");

`

var JS_LIST string = `debugLog(scriptName + " executing JS_LIST");

const allWindows = workspace.windowList();
for (let i = 0; i < allWindows.length; i++) {
    if ({{if .IncludeSpecialWindows}}true{{else}}!allWindows[i].specialWindow{{end}}) {
        let w = allWindows[i];
        returnResult(w.internalId + "\t" + w.resourceClass + "\t" + (w.resourceName.length == 0 ? "n/a" : w.resourceName ) {{if .ShowPids}}+ "\t" + w.pid{{end}}{{if .ShowCaptions}}+ "\t" + w.caption{{end}});
    }
}

`

var JS_FIND string = `debugLog(scriptName + " executing JS_SEARCH");

const allWindows = workspace.windowList();
let results = [];
let regExp;
try {
    regExp = new RegExp({{jsString .SearchTerm}}, 'i');
} catch (error) {
    returnError("Invalid regular expression: " + error.message);
}
if (regExp) {
    for (let i = 0; i < allWindows.length; i++) {
        let w = allWindows[i];
        if (w.{{.SearchField}}.search(regExp) >= 0) {
            results.push(w);
        }
    }
    for (let i = 0; i < results.length; i++) {
        returnResult(results[i].internalId);
    }
}

`

var JS_GET_ACTIVE_WINDOW string = `debugLog(scriptName + " executing JS_GET_ACTIVE_WINDOW");

const activeWindow = workspace.activeWindow;
if (activeWindow.specialWindow) {
    returnError("No active regular window");
} else {
    returnResult(activeWindow.internalId);
}

`

var JS_GET_WINDOW_GEOMETRY string = `debugLog(scriptName + " executing JS_GET_WINDOW_GEOMETRY");

const targetWindow = findWindow({{jsString .Uuid}});

if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    const result = Math.round(targetWindow.x) + " " + Math.round(targetWindow.y) + " " + targetWindow.width + " " + targetWindow.height;
    returnResult(result);
}

`

var JS_GET_WORKSPACE string = `debugLog(scriptName + " executing JS_GET_WORKSPACE");

returnResult(workspace.currentDesktop.x11DesktopNumber);
`

var JS_SET_WORKSPACE string = `debugLog(scriptName + " executing JS_SET_WORKSPACE");

let ws = workspace.desktops.find((ws) => ws.x11DesktopNumber == {{.WorkspaceId}});
if (ws) {
    workspace.currentDesktop = ws;
} else {
    returnError("Invalid workspace number: " + {{.WorkspaceId}});
}

`

var JS_ACTIVATE_WINDOW string = `debugLog(scriptName + " executing JS_ACTIVATE_WINDOW");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("Activating window with UUID=" + {{jsString .Uuid}});
    workspace.activeWindow = targetWindow;
}

`

var JS_SET_WINDOW_SIZE string = `debugLog(scriptName + " executing JS_SET_WINDOW_SIZE");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("New size for window with UUID=" + {{jsString .Uuid}} + ": width={{.Width}}, height={{.Height}}");
    updateWindowGeometry(targetWindow, {
        width: {{.Width}},
        height: {{.Height}},
    });
}

`

var JS_SET_WINDOW_POSITION string = `debugLog(scriptName + " executing JS_SET_WINDOW_POSITION");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("New position for window with UUID=" + {{jsString .Uuid}} + ": X={{.X}}, Y={{.Y}}");
    updateWindowGeometry(targetWindow, {
        x: {{.X}},
        y: {{.Y}}
    });
}

`

var JS_SET_WINDOW_GEOMETRY string = `debugLog(scriptName + " executing JS_SET_WINDOW_GEOMETRY");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("New geometry for window with UUID=" + {{jsString .Uuid}} + ": X={{.X}}, Y={{.Y}}, width={{.Width}}, height={{.Height}}");
    updateWindowGeometry(targetWindow, {
        width: {{.Width}},
        height: {{.Height}},
        x: {{.X}},
        y: {{.Y}}
    });
}

`

var JS_SET_WINDOW_WORKSPACE string = `debugLog(scriptName + " executing JS_SET_WINDOW_WORKSPACE");

const targetWindow = findWindow({{jsString .Uuid}});
const targetWorkspace = workspace.desktops.find(
    (desktop) => desktop.x11DesktopNumber == {{.WorkspaceId}}
);

if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else if (!targetWorkspace) {
    returnError("Invalid workspace number: " + {{.WorkspaceId}});
} else {
    targetWindow.desktops = [targetWorkspace];
}

`

var JS_SET_WINDOW_PROPERTY string = `debugLog(scriptName + " executing JS_SET_WINDOW_PROPERTY");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("Setting property (value={{.PropertyValue}}) {{.WindowProperty}} on window with UUID=" + {{jsString .Uuid}});
    targetWindow.{{.WindowProperty}} = {{if (eq .PropertyValue "toggle")}}!targetWindow.{{.WindowProperty}}{{else}}{{.PropertyValue}}{{end}};
}

`

var JS_CLOSE_WINDOW string = `debugLog(scriptName + " executing JS_CLOSE_WINDOW");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("Closing window with UUID=" + {{jsString .Uuid}});
    targetWindow.closeWindow();
}

`

var JS_MOUSE_POS string = `debugLog(scriptName + " executing JS_MOUSE_POS");

const x = workspace.cursorPos.x;
const y = workspace.cursorPos.y;

returnResult(x + " " + y);

`

var JS_LIST_OUTPUTS string = `debugLog(scriptName + " executing JS_LIST_OUTPUTS");

const outputs = workspace.screens;

for (let i = 0; i < outputs.length; i++) {
    const output = outputs[i];
    returnResult(output.name);
}

`

var JS_GET_ACTIVE_OUTPUT string = `debugLog(scriptName + " executing JS_GET_ACTIVE_OUTPUT");

returnResult(workspace.activeScreen.name);

`

var JS_GET_CURSOR_OUTPUT string = `debugLog(scriptName + " executing JS_GET_CURSOR_OUTPUT");

const output = workspace.screenAt(workspace.cursorPos);
if (output != null) {
    returnResult(output.name);
} else {
    returnError("Unable to determine the output with the mouse cursor.");
}

`

var JS_GET_ACTIVE_WINDOW_OUTPUT string = `debugLog(scriptName + " executing JS_GET_ACTIVE_WINDOW_OUTPUT");

const window = workspace.activeWindow;

if (window != null) {
    returnResult(window.output.name);
} else {
    returnError("Unable to determine the active window");
}

`

var JS_GET_OUTPUT_GEOMETRY string = `debugLog(scriptName + " executing JS_OUTPUT_GEOMETRY");

const outputName = {{jsString .OutputName}};

const output = findOutput(outputName);

if (!output) {
    returnError("Output not found: " + outputName);
} else {
    let area;
    {{if .ClientArea}}
        const desktop = workspace.currentDesktopForScreen(output);
        area = workspace.clientArea(KWin.MaximizeArea, output, desktop);
    {{else}}
        area = output.geometry;
    {{end}}
    returnResult(
        Math.round(area.x) + " " +
        Math.round(area.y) + " " +
        Math.round(area.width) + " " +
        Math.round(area.height)
    );
}

`

var JS_GET_WINDOW_OPACITY string = `debugLog(scriptName + " executing JS_GET_WINDOW_OPACITY");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    returnResult(targetWindow.opacity);
}

`

var JS_SET_WINDOW_OPACITY string = `debugLog(scriptName + " executing JS_SET_WINDOW_OPACITY");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("Setting opacity of window with UUID=" + {{jsString .Uuid}} + " to {{.Opacity}}");
    targetWindow.opacity = {{.Opacity}};
}

`

var JS_INCREASE_WINDOW_OPACITY string = `debugLog(scriptName + " executing JS_INCREASE_WINDOW_OPACITY");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    let newOpacity = targetWindow.opacity;
    newOpacity += 0.05;
    if (newOpacity > 1.0) {
        newOpacity = 1.0;
    }
    debugLog("Setting opacity of window with UUID=" + {{jsString .Uuid}} + " to " + newOpacity);
    targetWindow.opacity = newOpacity;
}

`

var JS_DECREASE_WINDOW_OPACITY string = `debugLog(scriptName + " executing JS_DECREASE_WINDOW_OPACITY");

const targetWindow = findWindow({{jsString .Uuid}});
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    let newOpacity = targetWindow.opacity;
    newOpacity -= 0.05;
    if (newOpacity < 0.1) {
        newOpacity = 0.1;
    }
    debugLog("Setting opacity of window with UUID=" + {{jsString .Uuid}} + " to " + newOpacity);
    targetWindow.opacity = newOpacity;
}

`

var JS_SET_WINDOW_GEOMETRY_RELATIVE string = `debugLog(scriptName + " executing JS_SET_WINDOW_GEOMETRY_RELATIVE");

const targetWindow = findWindow({{jsString .Uuid}});
if (targetWindow) {
    const area = clientAreaForWindow(targetWindow);

    const width = Math.round(area.width * {{.RelativeWidth}} / 100);
    const height = Math.round(area.height * {{.RelativeHeight}} / 100);
    const x = area.x + Math.round(area.width * {{.RelativeX}} / 100);
    const y = area.y + Math.round(area.height * {{.RelativeY}} / 100);

    debugLog("Setting geometry of window with UUID=" + {{jsString .Uuid}} +
        " to: X=" + x +
        " Y=" + y +
        " width=" + width +
        " height=" + height);
    updateWindowGeometry(targetWindow, {
        x,
        y,
        width,
        height
    });
} else {
    returnError("Window not found: " + {{jsString .Uuid}})
}

`

var JS_SET_WINDOW_SIZE_RELATIVE string = `debugLog(scriptName + " executing JS_SET_WINDOW_SIZE_RELATIVE");

const targetWindow = findWindow({{jsString .Uuid}});
if (targetWindow) {
    const area = clientAreaForWindow(targetWindow);

    const width = Math.round(area.width * {{.RelativeWidth}} / 100);
    const height = Math.round(area.height * {{.RelativeHeight}} / 100);

    debugLog("Setting size of window with UUID=" + {{jsString .Uuid}} +
        " to: width=" + width +
        " height=" + height);
    updateWindowGeometry(targetWindow, {
        width,
        height
    });
} else {
    returnError("Window not found: " + {{jsString .Uuid}})
}

`

var JS_SET_WINDOW_POSITION_RELATIVE string = `debugLog(scriptName + " executing JS_SET_WINDOW_POSITION_RELATIVE");

const targetWindow = findWindow({{jsString .Uuid}});
if (targetWindow) {
    const area = clientAreaForWindow(targetWindow);

    const x = area.x + Math.round(area.width * {{.RelativeX}} / 100);
    const y = area.y + Math.round(area.height * {{.RelativeY}} / 100);

    debugLog("Setting position of window with UUID=" + {{jsString .Uuid}} +
        " to: X=" + x +
        " Y=" + y);
    updateWindowGeometry(targetWindow, {
        x,
        y
    });
} else {
    returnError("Window not found: " + {{jsString .Uuid}})
}

`

var JS_LIST_TILES string = `debugLog(scriptName + " executing JS_LIST_TILES");

let output;
const outputName = {{jsString .OutputName}};

if (outputName === "") {
    output = workspace.activeScreen;
} else {
    output = findOutput(outputName);
}

if (!output) {
    returnError("Output not found: " + outputName);
} else {
    const tileManager = workspace.tilingForScreen(output);
    const rootTile = tileManager.rootTile;
    const rows = [];

    rows.push([
        "OUTPUT",
        "PATH",
        "TYPE",
        "RELATIVE",
        "ABSOLUTE",
        "WINDOWS"
    ].join("\t"));

    enumerateTiles(rootTile, {{.LeavesOnly}}, (tile, path, isLeaf) => {
        const relative = tile.relativeGeometry;
        const absolute = tile.absoluteGeometryInScreen;

        rows.push([
            output.name,
            path,
            isLeaf ? "Leaf" : "Layout",
            [
                formatFloat(relative.x),
                formatFloat(relative.y),
                formatFloat(relative.width),
                formatFloat(relative.height),
            ].join(" "),
            [
                absolute.x,
                absolute.y,
                absolute.width,
                absolute.height,
            ].join(" "),
            tile.windows.length,
        ].join("\t"));
    });
    if (rows.length > 1) {
        returnResult(rows.join("\n"));
    }
}

`

var JS_GET_WINDOW_TILE string = `debugLog(scriptName + " executing JS_GET_WINDOW_TILE");

const targetWindow = findWindow({{jsString .Uuid}});

if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    const output = targetWindow.output;
    const currentDesktop = workspace.currentDesktopForScreen(output);
    const windowDesktops = targetWindow.desktops;

    let tileDesktop;

    if (windowDesktops.length === 1) {
        // window.tile can retain this desktop's association while it is inactive
        tileDesktop = windowDesktops[0];
    } else if (
        windowDesktops.length === 0 || // window is on all desktops
        windowDesktops.includes(currentDesktop)
    ) {
        // KWin selects this desktop's association when it becomes active
        tileDesktop = currentDesktop;
    } else {
        // Multi-desktop window is hidden on the current desktop.
        // window.tile may contain a retained association, but its desktop
        // cannot be identified through the public scripting API.
        returnError("Cannot determine the virtual desktop for the window's tile");
    }

    if (tileDesktop) {
        const windowTile = targetWindow.tile;
        if (windowTile === null) {
            returnError("It appears that window " + {{jsString .Uuid}} + " is not tiled");
        } else if (typeof targetWindow.tile.layoutDirection === "undefined") {
            returnError("Window " + {{jsString .Uuid}} + " is assigned to a quick tile");
        } else {
            const pathInfo = pathForTile(windowTile);
            if (pathInfo === null) {
                returnError("Unable to determine tile path");
            } else {
                const { path } = pathInfo;
                const relative = windowTile.relativeGeometry;
                const absolute = windowTile.absoluteGeometryInScreen;
                const rows = [];
                rows.push([
                    "OUTPUT",
                    "DESKTOP",
                    "PATH",
                    "RELATIVE",
                    "ABSOLUTE",
                ].join("\t"));
                rows.push([
                    output.name,
                    tileDesktop.x11DesktopNumber,
                    path,
                    [
                        formatFloat(relative.x),
                        formatFloat(relative.y),
                        formatFloat(relative.width),
                        formatFloat(relative.height),
                    ].join(" "),
                    [
                        absolute.x,
                        absolute.y,
                        absolute.width,
                        absolute.height,
                    ].join(" "),
                ].join("\t"));
                returnResult(rows.join("\n"));
            }
        }
    }
}

`

var JS_SET_WINDOW_TILE string = `debugLog(scriptName + " executing JS_SET_WINDOW_TILE");

const windowId = {{jsString .Uuid}};
const outputName = {{jsString .OutputName}};
const targetWindow = findWindow(windowId);
const tilePath = {{jsString .TilePath}};

if (!targetWindow) {
    returnError("Window not found: " + windowId);
} else {
    let targetOutput;
    if (outputName === "") {
        targetOutput = targetWindow.output;
    } else {
        targetOutput = findOutput(outputName);
    }
    if (!targetOutput) {
        returnError("Output not found: " + outputName);
    } else {
        const targetDesktop = workspace.currentDesktopForScreen(targetOutput);
        const rootTile = workspace.tilingForScreen(targetOutput).rootTile;
        const targetTile = resolveTile(rootTile, tilePath);
        if (!targetTile) {
            returnError("Unable to find target tile: " + tilePath);
        } else {
            if (!isOnCurrentDesktop(targetWindow, targetOutput)) {
                targetWindow.desktops = [targetDesktop];
            }
            if (!assignToTile(targetWindow, targetTile)) {
                returnError("Unable to tile window " + windowId + " to tile " + tilePath);
            }
        }
    }
}

`

var JS_FOOTER string = `close();
debugLog(scriptName + " END");
`
