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

// return "true" if a window caption matches the case-insensitive regular
// expression supplied as parameter 1
const allWindows = workspace.windowList();
let regExp;
try {
    regExp = new RegExp({{jsString .P1}}, 'i');
} catch (error) {
    returnError("Invalid regular expression: " + error.message);
}
let result = false;
if (regExp) {
    for (let i = 0; i < allWindows.length; i++) {
        const w = allWindows[i];
        if (w.caption.search(regExp) >= 0) {
            result = true;
        }
    }
    returnResult(result);
}

close();
debugLog(scriptName + " END");
