# KWST Previous Window KWin script

This optional resident KWin script tracks regular window activation in memory
and provides the shortcut **[KWST] Switch to previously active window**. No key
sequence is assigned by default.

> [!IMPORTANT]
> Installing, reinstalling, or uninstalling the script does not take effect
> immediately. Log out of the Plasma session and log in again after performing
> any of these operations.

Install or replace the user-local package with:

```sh
./install.sh
```

After installation, enable **KWST Previous Window** under **System Settings →
Window Management → KWin Scripts**. Then assign a key sequence to the shortcut
under **System Settings → Keyboard → Shortcuts → KWin**.

Remove the user-local package with:

```sh
./uninstall.sh
```
