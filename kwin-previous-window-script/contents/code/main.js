// SPDX-FileCopyrightText: 2026 Pavel Urusov
// SPDX-License-Identifier: GPL-2.0-or-later

const activationHistory = [];

function sameWindow(first, second) {
    return first.internalId.toString() === second.internalId.toString();
}

function isTrackable(window) {
    return window && !window.specialWindow;
}

function removeFromHistory(window) {
    if (!window) {
        return;
    }

    for (let index = activationHistory.length - 1; index >= 0; index--) {
        if (sameWindow(activationHistory[index], window)) {
            activationHistory.splice(index, 1);
        }
    }
}

function recordActivation(window) {
    if (!isTrackable(window)) {
        return;
    }

    removeFromHistory(window);
    activationHistory.push(window);
}

function pruneHistory() {
    const windows = workspace.windowList();
    for (let index = activationHistory.length - 1; index >= 0; index--) {
        const window = activationHistory[index];
        const stillExists = windows.some((candidate) => sameWindow(candidate, window));
        if (!isTrackable(window) || !stillExists) {
            activationHistory.splice(index, 1);
        }
    }
}

function switchToPreviouslyActiveWindow() {
    recordActivation(workspace.activeWindow);
    pruneHistory();

    const activeWindow = workspace.activeWindow;
    for (let index = activationHistory.length - 1; index >= 0; index--) {
        const candidate = activationHistory[index];
        if (!isTrackable(activeWindow) || !sameWindow(candidate, activeWindow)) {
            workspace.activeWindow = candidate;
            return;
        }
    }
}

workspace.windowActivated.connect(recordActivation);
workspace.windowRemoved.connect(removeFromHistory);
recordActivation(workspace.activeWindow);

const shortcutName = "[KWST] Switch to previously active window";
registerShortcut(shortcutName, shortcutName, "", switchToPreviouslyActiveWindow);
