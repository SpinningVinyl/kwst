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

// your code goes here

close();
debugLog(scriptName + " END");
