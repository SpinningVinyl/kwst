package main

var JS_HEADER string = `
const dbusAddr = "{{.DbusAddr}}";

const debug = {{.Debug}};

const scriptName = "{{.ScriptName}}";

const debugLog = (msg) => {
    if (debug) {
        print(msg.toString());
    }
}

const close = () => {
    debugLog("Calling Close() on " + dbusAddr); 
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Close");
}

const returnResult = (msgBody) => {
    debugLog("RESULT: " + msgBody);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Msg", "result", msgBody.toString());
}

const returnError = (msgBody) => {
    debugLog("ERROR: " + msgBody);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Msg", "error", msgBody.toString());
}


debugLog(scriptName + " START");

`

var JS_LIST string = `debugLog(scriptName + " executing JS_LIST");

const allWindows = workspace.windowList();
for (let i = 0; i < allWindows.length; i++) {
    if ({{if .IncludeSpecialWindows }}true{{else}}!allWindows[i].specialWindow{{end}}) {
        returnResult(allWindows[i].internalId + "\t" + allWindows[i].resourceClass + "\t" + allWindows[i].resourceName + "\t" + allWindows[i].pid);
    }
}

`

var JS_FIND string = `debugLog(scriptName + " executing JS_SEARCH");

const allWindows = workspace.windowList();
let results = [];
` + "const regExp = new RegExp(String.raw`{{.SearchTerm}}`, 'i');\n" +
	`for (let i = 0; i < allWindows.length; i++) {
    let w = allWindows[i];
    if (w.{{.SearchField}}.search(regExp) >= 0) {
        results.push(w);
    }
}
for (let i = 0; i < results.length; i++) {
    returnResult(results[i].internalId);
}

`

var JS_GET_ACTIVE_WINDOW string = `debugLog(scriptName + " executing JS_GET_ACTIVE_WINDOW");

returnResult(workspace.activeWindow.internalId);

`

var JS_GET_WINDOW_GEOMETRY string = `debugLog(scriptName + " executing JS_GET_WINDOW_GEOMETRY");

const allWindows = workspace.windowList();
for (let i = 0; i < allWindows.length; i++) {
    let w = allWindows[i];
    if (w.internalId == "{{.Uuid}}") {
         let result = w.x + " " + w.y + " " + w.width + " " + w.height;
         returnResult(result);
    }
}

`

var JS_GET_WORKSPACE string = `debugLog(scriptName + " executing JS_GET_WORKSPACE");

returnResult(workspace.currentDesktop.x11DesktopNumber);
`


var JS_FOOTER string = `close();
debugLog(scriptName + " END");
`
