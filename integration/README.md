# Live KWin integration tests

These integration tests build and run the real `kwst` executable against the current
KWin session. Since they involve creating and manipulating windows, as well as switching 
virtual desktops, it is advised not to run them during important interactive work.

## Prerequisites

- A running KWin session w/session D-Bus.
- KDialog available in `PATH`.
- Go 1.23 or later.

## Running

From the repository root:

```sh
KWST_INTEGRATION=1 go test -tags=integration -count=1 -timeout=2m ./integration
```

The test uses KDialog windows as test fixtures. It only manipulates
windows that the test harness creates. If the session has only
one virtual desktop, the test creates a uniquely named temporary desktop
and removes it after restoring the original desktop.
