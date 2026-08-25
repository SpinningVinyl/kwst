// SPDX-License-Identifier: GPL-2.0-or-later

const dbusAddr = "{{.DbusAddr}}";

const debug = {{.Debug}};

const scriptName = "{{.ScriptName}}";

let exitCode = 0;

const results = [];
const errors = [];

// use this function to print debug messages to the system log.
// the messages will be printed only if kwst is called with the `--debug` flag.
// to make sure that debug messages are added to the system log, 
// you should run `kdebugsettings` and set KWin Scripting to `Full Debug`.
// you can read the messages by using the following command:
// `journalctl -f QT_CATEGORY=js QT_CATEGORY=kwin_scripting`
const debugLog = (msg) => {
    if (debug) {
        print(msg.toString());
    }
}

// this function tells kwst that the script is done and terminates the execution
// of the program.
const close = () => {
    debugLog("Calling Complete() on " + dbusAddr);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Complete", exitCode, results.join("\n"), errors.join("\n"));
}

// use this function to return results back to kwst. The results
// will be printed to stdout
const returnResult = (msgBody) => {
    debugLog("RESULT: " + msgBody);
    results.push(msgBody.toString());
}

// use this function to return errors back to kwst. The errors
// will be printed to stderr and kwst will exit with a non-zero status.
const returnError = (msgBody) => {
    exitCode = 1;
    debugLog("ERROR: " + msgBody);
    errors.push("KWin script returned an error: " + msgBody.toString());
}


const execute = () => {
    debugLog(scriptName + " START");

    // your code goes here
    // When inserting a parameter as a JavaScript string, use {{jsString .P1}}.
    // jsString includes the surrounding quotes, so do not add another pair.

}

// wrapping the call to your code in a try...catch...finally block ensures
// that if it throws, kwst is going to give you a meaningful error message
// instead of a generic timeout.
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
