// SPDX-License-Identifier: GPL-2.0-or-later

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

const execute = () => {
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
}

try {
    execute();
} catch (error) {
    const message = String(error);
    const stack = error && error.stack ? String(error.stack) : "";
    returnError(
        "Error executing KWin script: " + message +
        (stack ? "\n" + stack : "")
    );
} finally {
    close();
    debugLog(scriptName + " END");
}
