const dbusAddr = "{{.DbusAddr}}";

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

// return UUIDs of all fullscreen windows
const allWindows = workspace.windowList();
for (let i = 0; i < allWindows.length; i++) {
    const w = allWindows[i];
    if (w.fullScreen) {
        returnResult(w.internalId);
    }
}

close();
debugLog(scriptName + " END");
