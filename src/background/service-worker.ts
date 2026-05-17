import {
  getExtensionState,
  hasChromeStorage,
  setExtensionState,
  updateCurrentIndex,
  type ExtensionState,
  type UserSettings,
} from "../lib/extension/storage";
import type { Flashcard } from "../lib/types/flashcard";

const NOTIFICATION_ICON_PATHS = ["icon128.png", "logo.png"] as const;
const NOTIFICATION_ALARM_NAME = "jlpt-card-reminder";
const NOTIFICATION_ID_PREFIX = "jlpt-card-reminder";
const STORAGE_STATE_KEY = "jlptExtensionState";
const LEGACY_SETTINGS_KEY = "jlptSettings";
const LEGACY_PAUSED_KEY = "jlptNotificationPaused";

const CATEGORY_LABELS: Record<Flashcard["category"], string> = {
  gramma: "Ngữ pháp",
  locabulary: "Từ vựng",
  kanji: "Kanji",
  reading: "Đọc hiểu",
  listening: "Nghe hiểu",
};

if (typeof chrome !== "undefined") {
  void initializeExtensionState();

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
      if (areaName !== "local" || !didNotificationScheduleChange(changes))
        return;

      void refreshNotificationAlarm();
    });
  }

  if (chrome.runtime?.onMessage) {
    chrome.runtime.onMessage.addListener(
      (message: { type?: string }, _sender, sendResponse) => {
        if (message?.type !== "SHOW_STUDY_NOTIFICATION_NOW") return false;

        void showStudyNotification()
          .then((result) => sendResponse(result))
          .catch((error) =>
            sendResponse({
              ok: false,
              error:
                error instanceof Error
                  ? error.message
                  : "Không thể hiển thị notification.",
            }),
          );
        return true;
      },
    );
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

  if (
    !currentState.settings.notificationEnabled ||
    currentState.notificationPaused
  )
    return;

  await createNotificationAlarm(
    currentState.settings.notificationIntervalMinutes,
  );
}

async function showStudyNotification() {
  if (!chrome.notifications)
    return { ok: false, error: "Chrome notifications API không khả dụng." };

  const permissionLevel = await getNotificationPermissionLevel();
  if (permissionLevel === "denied") {
    return {
      ok: false,
      error:
        "Chrome hoặc hệ điều hành đang chặn notification cho extension này.",
    };
  }

  const state = await getExtensionState();
  if (!state.settings.notificationEnabled || state.notificationPaused) {
    await clearNotificationAlarm();
    return { ok: false, error: "Notification đang tắt hoặc đang tạm dừng." };
  }

  const cards = getEligibleCards(state);
  if (cards.length === 0) {
    return {
      ok: false,
      error: "Không có flashcard phù hợp với level/category đang chọn.",
    };
  }

  const notificationIndex = getNotificationIndex(
    cards,
    state.currentIndex,
    state.settings.orderMode,
  );
  const card = cards[notificationIndex];
  const notificationId = `${NOTIFICATION_ID_PREFIX}-${Date.now()}`;

  const result = await createNotification(notificationId, card);
  if (!result.ok) return result;

  const nextIndex = getNextCurrentIndex(
    cards.length,
    notificationIndex,
    state.settings.orderMode,
  );
  await updateCurrentIndex(nextIndex);

  return result;
}

function didNotificationScheduleChange(
  changes: Record<string, chrome.storage.StorageChange>,
) {
  const stateChange = changes[STORAGE_STATE_KEY];
  const oldState = stateChange?.oldValue as Partial<ExtensionState> | undefined;
  const newState = stateChange?.newValue as Partial<ExtensionState> | undefined;

  if (
    stateChange &&
    didSettingsScheduleChange(oldState?.settings, newState?.settings)
  ) {
    return true;
  }

  if (
    stateChange &&
    oldState?.notificationPaused !== newState?.notificationPaused
  ) {
    return true;
  }

  const settingsChange = changes[LEGACY_SETTINGS_KEY];
  const oldSettings = settingsChange?.oldValue as
    | Partial<UserSettings>
    | undefined;
  const newSettings = settingsChange?.newValue as
    | Partial<UserSettings>
    | undefined;

  if (settingsChange && didSettingsScheduleChange(oldSettings, newSettings)) {
    return true;
  }

  const pausedChange = changes[LEGACY_PAUSED_KEY];
  return Boolean(
    pausedChange && pausedChange.oldValue !== pausedChange.newValue,
  );
}

function didSettingsScheduleChange(
  oldSettings?: Partial<UserSettings>,
  newSettings?: Partial<UserSettings>,
) {
  return (
    oldSettings?.notificationEnabled !== newSettings?.notificationEnabled ||
    oldSettings?.notificationIntervalMinutes !==
      newSettings?.notificationIntervalMinutes
  );
}

function getEligibleCards(state: ExtensionState) {
  return state.dataset.filter(
    (card) =>
      state.settings.selectedLevels.includes(card.level) &&
      state.settings.enabledCategories.includes(card.category),
  );
}

function getNotificationIndex(
  cards: Flashcard[],
  currentIndex: number,
  orderMode: UserSettings["orderMode"],
) {
  if (orderMode === "random") return Math.floor(Math.random() * cards.length);

  return normalizeIndex(currentIndex, cards.length);
}

function getNextCurrentIndex(
  cardCount: number,
  notificationIndex: number,
  orderMode: UserSettings["orderMode"],
) {
  if (cardCount < 2) return 0;
  if (orderMode === "sequential") return (notificationIndex + 1) % cardCount;

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

async function createNotification(notificationId: string, card: Flashcard) {
  const title = formatNotificationTitle(card);
  const message = [`Nghĩa: ${card.mean}`, card.example]
    .filter(Boolean)
    .join("\n");

  let lastError: string | undefined;

  for (const iconPath of NOTIFICATION_ICON_PATHS) {
    const result = await createNotificationWithIcon(
      notificationId,
      iconPath,
      title,
      message,
    );

    if (result.ok) return result;

    lastError = result.error;
  }

  return {
    ok: false,
    error:
      lastError ??
      "Chrome không thể tải icon PNG cho notification từ extension.",
  };
}

function createNotificationWithIcon(
  notificationId: string,
  iconPath: (typeof NOTIFICATION_ICON_PATHS)[number],
  title: string,
  message: string,
) {
  return new Promise<{ ok: boolean; notificationId?: string; error?: string }>(
    (resolve) => {
      chrome.notifications.create(
        notificationId,
        {
          type: "basic",
          iconUrl: chrome.runtime.getURL(iconPath),
          title,
          message,
          priority: 2,
        },
        (createdNotificationId) => {
          const errorMessage = chrome.runtime?.lastError?.message;

          if (errorMessage) {
            console.warn(
              `JLPT notification failed with ${iconPath}:`,
              errorMessage,
            );
            resolve({ ok: false, error: errorMessage });
            return;
          }

          if (!createdNotificationId) {
            resolve({
              ok: false,
              error: "Chrome không trả về notification id.",
            });
            return;
          }

          resolve({ ok: true, notificationId: createdNotificationId });
        },
      );
    },
  );
}

function formatNotificationTitle(card: Flashcard) {
  const levelLabel = card.level.toUpperCase();
  const categoryLabel = CATEGORY_LABELS[card.category];
  const reading = card.hiragana ? ` (${card.hiragana})` : "";

  return `[${levelLabel} - ${categoryLabel}]  ${card.name}${reading}`;
}

function createNotificationAlarm(intervalMinutes: number) {
  return new Promise<void>((resolve) => {
    const normalizedInterval = Math.max(0.5, intervalMinutes);

    chrome.alarms.create(
      NOTIFICATION_ALARM_NAME,
      {
        delayInMinutes: normalizedInterval,
        periodInMinutes: normalizedInterval,
      },
      () => {
        if (chrome.runtime?.lastError) {
          console.warn(
            "JLPT notification alarm failed:",
            chrome.runtime.lastError.message,
          );
        }
        resolve();
      },
    );
  });
}

function clearNotificationAlarm() {
  return new Promise<boolean>((resolve) => {
    chrome.alarms.clear(NOTIFICATION_ALARM_NAME, resolve);
  });
}

function getNotificationPermissionLevel() {
  return new Promise<"granted" | "denied" | undefined>((resolve) => {
    if (!chrome.notifications?.getPermissionLevel) {
      resolve(undefined);
      return;
    }

    chrome.notifications.getPermissionLevel((level) => {
      resolve(level);
    });
  });
}
