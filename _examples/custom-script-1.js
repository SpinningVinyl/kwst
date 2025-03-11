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

// return UUIDs of all fullscreen windows
const allWindows = workspace.windowList();
for (let i = 0; i < allWindows.length; i++) {
    let w = allWindows[i];
    if (w.fullScreen == true) {
         returnResult(w.internalId);
    }
}

close();
debugLog(scriptName + " END");
