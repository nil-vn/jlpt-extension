import type { Flashcard, FlashcardCategory, JlptLevel, StudySettings } from '../types/flashcard';

export type UserSettings = StudySettings & {
  theme: 'light' | 'dark' | 'system';
  notificationIntervalMinutes: number;
  notificationEnabled: boolean;
  orderMode: 'random' | 'sequential';
  notificationDisplaySeconds?: number;
};

export type ExtensionState = {
  dataset: Flashcard[];
  currentIndex: number;
  bookmarkedCardIds: string[];
  notesByCardId: Record<string, string>;
  settings: UserSettings;
  notificationPaused?: boolean;
};

type StoredExtensionState = Partial<ExtensionState>;
type StorageSnapshot = {
  jlptExtensionState?: StoredExtensionState;
  jlptDataset?: Flashcard[];
  dataset?: Flashcard[];
  jlptCurrentIndex?: number;
  jlptBookmarkedCardIds?: string[];
  jlptNotesByCardId?: Record<string, string>;
  jlptSettings?: Partial<UserSettings>;
  jlptNotificationPaused?: boolean;
};

const EXTENSION_STATE_KEY = 'jlptExtensionState';
const STORAGE_KEYS: Array<keyof StorageSnapshot> = [
  EXTENSION_STATE_KEY,
  'jlptDataset',
  'dataset',
  'jlptCurrentIndex',
  'jlptBookmarkedCardIds',
  'jlptNotesByCardId',
  'jlptSettings',
  'jlptNotificationPaused'
];

export const DEFAULT_SETTINGS: UserSettings = {
  dailyGoal: 20,
  selectedLevels: ['n5', 'n4'],
  enabledCategories: ['locabulary', 'kanji', 'gramma'],
  theme: 'system',
  notificationIntervalMinutes: 60,
  notificationEnabled: false,
  orderMode: 'sequential',
  notificationDisplaySeconds: 20
};

export const DEFAULT_EXTENSION_STATE: ExtensionState = {
  dataset: [],
  currentIndex: 0,
  bookmarkedCardIds: [],
  notesByCardId: {},
  settings: DEFAULT_SETTINGS,
  notificationPaused: !DEFAULT_SETTINGS.notificationEnabled
};

let devState: ExtensionState = cloneState(DEFAULT_EXTENSION_STATE);

export function hasChromeStorage() {
  return typeof globalThis.chrome !== 'undefined' && Boolean(globalThis.chrome.storage?.local);
}

export async function getExtensionState(): Promise<ExtensionState> {
  if (!hasChromeStorage()) return cloneState(devState);

  const stored = (await globalThis.chrome.storage.local.get(STORAGE_KEYS)) as StorageSnapshot;
  return normalizeStoredState(stored);
}

export async function setExtensionState(state: ExtensionState): Promise<void> {
  const nextState = cloneState(normalizeState(state));
  devState = nextState;

  if (!hasChromeStorage()) return;

  await globalThis.chrome.storage.local.set(toStorageItems(nextState));
}

export async function updateSettings(settings: Partial<UserSettings>): Promise<ExtensionState> {
  const state = await getExtensionState();
  const nextState = normalizeState({
    ...state,
    settings: {
      ...state.settings,
      ...settings
    },
    notificationPaused:
      typeof settings.notificationEnabled === 'boolean' ? !settings.notificationEnabled : state.notificationPaused
  });

  await setExtensionState(nextState);
  return nextState;
}

export async function saveDataset(dataset: Flashcard[]): Promise<ExtensionState> {
  const state = await getExtensionState();
  const nextState = normalizeState({
    ...state,
    dataset,
    currentIndex: 0
  });

  await setExtensionState(nextState);
  return nextState;
}

export async function saveNote(cardId: string, note: string): Promise<ExtensionState> {
  if (!cardId) return getExtensionState();

  const state = await getExtensionState();
  const notesByCardId = { ...state.notesByCardId };

  if (note.trim().length === 0) {
    delete notesByCardId[cardId];
  } else {
    notesByCardId[cardId] = note;
  }

  const nextState = normalizeState({ ...state, notesByCardId });
  await setExtensionState(nextState);
  return nextState;
}

export async function toggleBookmark(cardId: string): Promise<ExtensionState> {
  if (!cardId) return getExtensionState();

  const state = await getExtensionState();
  const isBookmarked = state.bookmarkedCardIds.includes(cardId);
  const bookmarkedCardIds = isBookmarked
    ? state.bookmarkedCardIds.filter((bookmarkedCardId) => bookmarkedCardId !== cardId)
    : [...state.bookmarkedCardIds, cardId];
  const nextState = normalizeState({ ...state, bookmarkedCardIds });

  await setExtensionState(nextState);
  return nextState;
}

export async function updateCurrentIndex(currentIndex: number): Promise<ExtensionState> {
  const state = await getExtensionState();
  const nextState = normalizeState({ ...state, currentIndex });

  await setExtensionState(nextState);
  return nextState;
}

export async function setNotificationPaused(notificationPaused: boolean): Promise<ExtensionState> {
  const state = await getExtensionState();
  const nextState = normalizeState({
    ...state,
    notificationPaused,
    settings: {
      ...state.settings,
      notificationEnabled: !notificationPaused
    }
  });

  await setExtensionState(nextState);
  return nextState;
}

function normalizeStoredState(stored: StorageSnapshot): ExtensionState {
  return normalizeState({
    ...stored.jlptExtensionState,
    dataset: stored.jlptExtensionState?.dataset ?? stored.jlptDataset ?? stored.dataset ?? [],
    currentIndex: stored.jlptExtensionState?.currentIndex ?? stored.jlptCurrentIndex ?? 0,
    bookmarkedCardIds: stored.jlptExtensionState?.bookmarkedCardIds ?? stored.jlptBookmarkedCardIds ?? [],
    notesByCardId: stored.jlptExtensionState?.notesByCardId ?? stored.jlptNotesByCardId ?? {},
    settings: {
      ...DEFAULT_SETTINGS,
      ...stored.jlptSettings,
      ...stored.jlptExtensionState?.settings
    },
    notificationPaused:
      stored.jlptExtensionState?.notificationPaused ??
      stored.jlptNotificationPaused ??
      ((stored.jlptExtensionState?.settings ?? stored.jlptSettings)?.notificationEnabled === false)
  });
}

function normalizeState(state: StoredExtensionState): ExtensionState {
  const settings = normalizeSettings(state.settings);
  const dataset = Array.isArray(state.dataset) ? state.dataset : [];
  const maxIndex = Math.max(dataset.length - 1, 0);
  const currentIndex = clampNumber(state.currentIndex, 0, 0, maxIndex);

  return {
    dataset,
    currentIndex,
    bookmarkedCardIds: Array.isArray(state.bookmarkedCardIds) ? Array.from(new Set(state.bookmarkedCardIds)) : [],
    notesByCardId: isRecord(state.notesByCardId) ? state.notesByCardId : {},
    settings,
    notificationPaused:
      typeof state.notificationPaused === 'boolean' ? state.notificationPaused : !settings.notificationEnabled
  };
}

function normalizeSettings(settings: Partial<UserSettings> | undefined): UserSettings {
  const merged = { ...DEFAULT_SETTINGS, ...settings };

  return {
    ...merged,
    dailyGoal: clampNumber(merged.dailyGoal, DEFAULT_SETTINGS.dailyGoal, 1, 999),
    selectedLevels: normalizeLevels(merged.selectedLevels),
    enabledCategories: normalizeCategories(merged.enabledCategories),
    theme: normalizeTheme(merged.theme),
    notificationIntervalMinutes: clampNumber(
      merged.notificationIntervalMinutes,
      DEFAULT_SETTINGS.notificationIntervalMinutes,
      1,
      1440
    ),
    notificationEnabled: Boolean(merged.notificationEnabled),
    orderMode: merged.orderMode === 'random' ? 'random' : 'sequential',
    notificationDisplaySeconds: clampNumber(
      merged.notificationDisplaySeconds,
      DEFAULT_SETTINGS.notificationDisplaySeconds ?? 20,
      5,
      120
    )
  };
}

function normalizeLevels(levels: unknown): JlptLevel[] {
  if (!Array.isArray(levels)) return DEFAULT_SETTINGS.selectedLevels;

  const normalized = levels
    .map((level) => String(level).toLowerCase())
    .filter((level): level is JlptLevel => ['n5', 'n4', 'n3', 'n2', 'n1'].includes(level));

  return normalized.length > 0 ? Array.from(new Set(normalized)) : DEFAULT_SETTINGS.selectedLevels;
}

function normalizeCategories(categories: unknown): FlashcardCategory[] {
  if (!Array.isArray(categories)) return DEFAULT_SETTINGS.enabledCategories;

  const normalized = categories
    .map((category) => normalizeCategory(category))
    .filter((category): category is FlashcardCategory =>
      ['gramma', 'locabulary', 'kanji', 'reading', 'listening'].includes(category)
    );

  return normalized.length > 0 ? Array.from(new Set(normalized)) : DEFAULT_SETTINGS.enabledCategories;
}

function normalizeCategory(category: unknown) {
  const normalized = String(category).toLowerCase();

  if (normalized === 'grammar') return 'gramma';
  if (normalized === 'vocabulary') return 'locabulary';

  return normalized;
}

function normalizeTheme(theme: unknown): UserSettings['theme'] {
  return theme === 'light' || theme === 'dark' || theme === 'system' ? theme : DEFAULT_SETTINGS.theme;
}

function clampNumber(value: unknown, fallback: number, min: number, max: number) {
  const parsed = Number(value);

  if (!Number.isFinite(parsed)) return fallback;

  return Math.min(max, Math.max(min, Math.round(parsed)));
}

function isRecord(value: unknown): value is Record<string, string> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function toStorageItems(state: ExtensionState) {
  return {
    [EXTENSION_STATE_KEY]: state,
    jlptDataset: state.dataset,
    jlptCurrentIndex: state.currentIndex,
    jlptBookmarkedCardIds: state.bookmarkedCardIds,
    jlptNotesByCardId: state.notesByCardId,
    jlptSettings: state.settings,
    jlptNotificationPaused: state.notificationPaused
  };
}

function cloneState(state: ExtensionState): ExtensionState {
  return JSON.parse(JSON.stringify(state)) as ExtensionState;
}
