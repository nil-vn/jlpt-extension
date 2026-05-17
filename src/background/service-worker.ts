import {
  getExtensionState,
  hasChromeStorage,
  setExtensionState,
  updateCurrentIndex,
  type ExtensionState,
  type UserSettings
} from '../lib/extension/storage';
import type { Flashcard } from '../lib/types/flashcard';

const NOTIFICATION_ICON_URL =
  'data:image/svg+xml;charset=UTF-8,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128" viewBox="0 0 128 128"><rect width="128" height="128" rx="28" fill="#4f46e5"/><text x="64" y="78" text-anchor="middle" font-family="Arial,sans-serif" font-size="42" font-weight="700" fill="white">日</text></svg>'
  );

const NOTIFICATION_ALARM_NAME = 'jlpt-card-reminder';
const NOTIFICATION_ID_PREFIX = 'jlpt-card-reminder';
const STORAGE_STATE_KEY = 'jlptExtensionState';
const LEGACY_SETTINGS_KEY = 'jlptSettings';
const LEGACY_PAUSED_KEY = 'jlptNotificationPaused';

const CATEGORY_LABELS: Record<Flashcard['category'], string> = {
  gramma: 'Ngữ pháp',
  locabulary: 'Từ vựng',
  kanji: 'Kanji',
  reading: 'Đọc hiểu',
  listening: 'Nghe hiểu'
};

if (typeof chrome !== 'undefined') {
  if (chrome.runtime?.onInstalled) {
    chrome.runtime.onInstalled.addListener(() => {
      void initializeExtensionState();
    });
  }

  if (chrome.runtime?.onStartup) {
    chrome.runtime.onStartup.addListener(() => {
      void initializeExtensionState();
    });
  }

  if (chrome.storage?.onChanged) {
    chrome.storage.onChanged.addListener((changes, areaName) => {
      if (areaName !== 'local' || !didNotificationScheduleChange(changes)) return;

      void refreshNotificationAlarm();
    });
  }

  if (chrome.alarms?.onAlarm) {
    chrome.alarms.onAlarm.addListener((alarm) => {
      if (alarm.name !== NOTIFICATION_ALARM_NAME) return;

      void showStudyNotification();
    });
  }
}

async function initializeExtensionState() {
  if (!hasChromeStorage()) return;

  const state = await getExtensionState();
  await setExtensionState(state);
  await refreshNotificationAlarm(state);
}

async function refreshNotificationAlarm(state?: ExtensionState) {
  if (!chrome.alarms) return;

  const currentState = state ?? (await getExtensionState());
  await clearNotificationAlarm();

  if (!currentState.settings.notificationEnabled || currentState.notificationPaused) return;

  await createNotificationAlarm(currentState.settings.notificationIntervalMinutes);
}

async function showStudyNotification() {
  if (!chrome.notifications) return;

  const state = await getExtensionState();
  if (!state.settings.notificationEnabled || state.notificationPaused) {
    await clearNotificationAlarm();
    return;
  }

  const cards = getEligibleCards(state);
  if (cards.length === 0) return;

  const notificationIndex = getNotificationIndex(cards, state.currentIndex, state.settings.orderMode);
  const card = cards[notificationIndex];
  const notificationId = `${NOTIFICATION_ID_PREFIX}-${Date.now()}`;

  await createNotification(notificationId, card);

  const nextIndex = getNextCurrentIndex(cards.length, notificationIndex, state.settings.orderMode);
  await updateCurrentIndex(nextIndex);

  if (state.settings.notificationDisplaySeconds) {
    scheduleNotificationClear(notificationId, state.settings.notificationDisplaySeconds);
  }
}

function didNotificationScheduleChange(changes: Record<string, chrome.storage.StorageChange>) {
  const stateChange = changes[STORAGE_STATE_KEY];
  const oldState = stateChange?.oldValue as Partial<ExtensionState> | undefined;
  const newState = stateChange?.newValue as Partial<ExtensionState> | undefined;

  if (stateChange && didSettingsScheduleChange(oldState?.settings, newState?.settings)) {
    return true;
  }

  if (stateChange && oldState?.notificationPaused !== newState?.notificationPaused) {
    return true;
  }

  const settingsChange = changes[LEGACY_SETTINGS_KEY];
  const oldSettings = settingsChange?.oldValue as Partial<UserSettings> | undefined;
  const newSettings = settingsChange?.newValue as Partial<UserSettings> | undefined;

  if (settingsChange && didSettingsScheduleChange(oldSettings, newSettings)) {
    return true;
  }

  const pausedChange = changes[LEGACY_PAUSED_KEY];
  return Boolean(pausedChange && pausedChange.oldValue !== pausedChange.newValue);
}

function didSettingsScheduleChange(oldSettings?: Partial<UserSettings>, newSettings?: Partial<UserSettings>) {
  return (
    oldSettings?.notificationEnabled !== newSettings?.notificationEnabled ||
    oldSettings?.notificationIntervalMinutes !== newSettings?.notificationIntervalMinutes
  );
}

function getEligibleCards(state: ExtensionState) {
  return state.dataset.filter(
    (card) =>
      state.settings.selectedLevels.includes(card.level) && state.settings.enabledCategories.includes(card.category)
  );
}

function getNotificationIndex(cards: Flashcard[], currentIndex: number, orderMode: UserSettings['orderMode']) {
  if (orderMode === 'random') return Math.floor(Math.random() * cards.length);

  return normalizeIndex(currentIndex, cards.length);
}

function getNextCurrentIndex(cardCount: number, notificationIndex: number, orderMode: UserSettings['orderMode']) {
  if (cardCount < 2) return 0;
  if (orderMode === 'sequential') return (notificationIndex + 1) % cardCount;

  let nextIndex = notificationIndex;
  while (nextIndex === notificationIndex) {
    nextIndex = Math.floor(Math.random() * cardCount);
  }

  return nextIndex;
}

function normalizeIndex(index: number, length: number) {
  if (length <= 0) return 0;

  return Math.min(Math.max(Math.round(index), 0), length - 1);
}

function createNotification(notificationId: string, card: Flashcard) {
  const title = formatNotificationTitle(card);
  const message = [`[THỰC] ${card.mean}`, card.example].filter(Boolean).join('\n');

  return new Promise<string>((resolve) => {
    chrome.notifications.create(
      notificationId,
      {
        type: 'basic',
        iconUrl: NOTIFICATION_ICON_URL,
        title,
        message,
        priority: 1
      },
      (createdNotificationId) => resolve(createdNotificationId)
    );
  });
}

function formatNotificationTitle(card: Flashcard) {
  const levelLabel = card.level.toUpperCase();
  const categoryLabel = CATEGORY_LABELS[card.category];
  const reading = card.hiragana ? ` (${card.hiragana})` : '';

  return `[${levelLabel} - ${categoryLabel}]  ${card.name}${reading}`;
}

function createNotificationAlarm(intervalMinutes: number) {
  return new Promise<void>((resolve) => {
    chrome.alarms.create(NOTIFICATION_ALARM_NAME, {
      delayInMinutes: intervalMinutes,
      periodInMinutes: intervalMinutes
    });
    resolve();
  });
}

function clearNotificationAlarm() {
  return new Promise<boolean>((resolve) => {
    chrome.alarms.clear(NOTIFICATION_ALARM_NAME, resolve);
  });
}

function scheduleNotificationClear(notificationId: string, displaySeconds: number) {
  setTimeout(() => {
    chrome.notifications.clear(notificationId);
  }, displaySeconds * 1000);
}
