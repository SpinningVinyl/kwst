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

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
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

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("Activating window with UUID=" + {{jsString .Uuid}});
    workspace.activeWindow = targetWindow;
}

`

var JS_SET_WINDOW_SIZE string = `debugLog(scriptName + " executing JS_SET_WINDOW_SIZE");

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("New size for window with UUID=" + {{jsString .Uuid}} + ": width={{.Width}}, height={{.Height}}");
    const newGeometry = Object.assign({}, targetWindow.frameGeometry);
    newGeometry.width = {{.Width}};
    newGeometry.height = {{.Height}};
    targetWindow.frameGeometry = newGeometry;
}

`

var JS_SET_WINDOW_POSITION string = `debugLog(scriptName + " executing JS_SET_WINDOW_POSITION");

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("New position for window with UUID=" + {{jsString .Uuid}} + ": X={{.X}}, Y={{.Y}}");
    const newGeometry = Object.assign({}, targetWindow.frameGeometry);
    newGeometry.x = {{.X}};
    newGeometry.y = {{.Y}};
    targetWindow.frameGeometry = newGeometry;
}

`

var JS_SET_WINDOW_GEOMETRY string = `debugLog(scriptName + " executing JS_SET_WINDOW_GEOMETRY");

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("New geometry for window with UUID=" + {{jsString .Uuid}} + ": X={{.X}}, Y={{.Y}}, width={{.Width}}, height={{.Height}}");
    const newGeometry = Object.assign({}, targetWindow.frameGeometry);
    newGeometry.width = {{.Width}};
    newGeometry.height = {{.Height}};
    newGeometry.x = {{.X}};
    newGeometry.y = {{.Y}};
    targetWindow.frameGeometry = newGeometry;
}

`

var JS_SET_WINDOW_WORKSPACE string = `debugLog(scriptName + " executing JS_SET_WINDOW_WORKSPACE");

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
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

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
if (!targetWindow) {
    returnError("Window not found: " + {{jsString .Uuid}});
} else {
    debugLog("Setting property (value={{.PropertyValue}}) {{.WindowProperty}} on window with UUID=" + {{jsString .Uuid}});
    targetWindow.{{.WindowProperty}} = {{if (eq .PropertyValue "toggle")}}!targetWindow.{{.WindowProperty}}{{else}}{{.PropertyValue}}{{end}};
}

`

var JS_CLOSE_WINDOW string = `debugLog(scriptName + " executing JS_CLOSE_WINDOW");

const targetWindow = workspace.windowList().find(
    (window) => window.internalId == {{jsString .Uuid}}
);
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

var JS_FOOTER string = `close();
debugLog(scriptName + " END");
`
