const dbusAddr = "{{.DbusAddr}}";

const debug = {{.Debug}};

const scriptName = "{{.ScriptName}}";

let exitCode = 0;

const results = [];
const errors = [];

const debugLog = (msg) => {
    if (debug) {
        print(msg.toString());
    }
}

const close = () => {
    debugLog("Calling Complete() on " + dbusAddr);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Complete", exitCode, results.join("\n"), errors.join("\n"));
}

const returnResult = (msgBody) => {
    debugLog("RESULT: " + msgBody);
    results.push(msgBody.toString());
}

const returnError = (msgBody) => {
    exitCode = 1;
    debugLog("ERROR: " + msgBody);
    errors.push("KWin script returned an error: " + msgBody.toString());
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
