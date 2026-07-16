const dbusAddr = "{{.DbusAddr}}";

const debug = {{.Debug}};

const scriptName = "{{.ScriptName}}";

let exitCode = 0;

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

// this function should be called at the end of your script.
// it tells kwst that the script is done and terminates the execution
// of the program.
const close = () => {
    debugLog("Calling CloseWithStatus() on " + dbusAddr);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "CloseWithStatus", exitCode);
}

// use this function to return results back to kwst. The results
// will be printed to stdout
const returnResult = (msgBody) => {
    debugLog("RESULT: " + msgBody);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Msg", "result", msgBody.toString());
}

// use this function to return errors back to kwst. The errors
// will be printed to stderr and kwst will exit with a non-zero status.
const returnError = (msgBody) => {
    exitCode = 1;
    debugLog("ERROR: " + msgBody);
    callDBus(dbusAddr, "/net/prsv/kwst", "net.prsv.kwst", "Msg", "error", msgBody.toString());
}


debugLog(scriptName + " START");

// your code goes here

close();
debugLog(scriptName + " END");
