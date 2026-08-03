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

var JS_FOOTER string = `close();
debugLog(scriptName + " END");
`
