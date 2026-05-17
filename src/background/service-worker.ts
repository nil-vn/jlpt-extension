import { getExtensionState, hasChromeStorage, setExtensionState } from '../lib/extension/storage';

if (typeof chrome !== 'undefined' && chrome.runtime?.onInstalled) {
  chrome.runtime.onInstalled.addListener(() => {
    void initializeExtensionState();
  });
}

async function initializeExtensionState() {
  if (!hasChromeStorage()) return;

  const state = await getExtensionState();
  await setExtensionState(state);
}
