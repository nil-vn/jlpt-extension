<script lang="ts">
  import { onMount } from 'svelte';
  import '../app.css';
  import { LevelSelector } from '../lib/components';
  import { levelDescriptions } from '../lib/data';
  import type { Flashcard, FlashcardCategory, JlptLevel, StudySettings } from '../lib/types/flashcard';

  type StudyMode = 'random' | 'sequential';
  type ThemeMode = 'light' | 'dark' | 'system';
  type PlannedFlashcard = Omit<Partial<Flashcard>, 'level' | 'category' | 'example'> & {
    level: string;
    category: string;
    name?: string;
    mean?: string;
    hiragana?: string;
    image?: string | null;
    audio?: string | null;
    example?: string | null;
  };
  type StoredStudySettings = StudySettings & {
    orderMode: StudyMode;
    notificationEnabled: boolean;
    notificationIntervalMinutes: number;
    notificationStartTime: string;
    notificationDisplaySeconds: number;
    theme: ThemeMode;
  };
  type StorageSnapshot = {
    jlptDataset?: PlannedFlashcard[];
    dataset?: PlannedFlashcard[];
    jlptSettings?: Partial<StoredStudySettings>;
  };
  type DatasetValidation = {
    status: 'idle' | 'valid' | 'partial' | 'invalid';
    message: string;
    validCount: number;
    invalidCount: number;
    fileName?: string;
  };

  const storageKeys: Array<keyof StorageSnapshot> = ['jlptDataset', 'dataset', 'jlptSettings'];
  const defaultSettings: StoredStudySettings = {
    dailyGoal: 20,
    selectedLevels: ['N5', 'N4'],
    enabledCategories: ['vocabulary', 'kanji', 'grammar'],
    orderMode: 'sequential',
    notificationEnabled: false,
    notificationIntervalMinutes: 60,
    notificationStartTime: '09:00',
    notificationDisplaySeconds: 20,
    theme: 'system'
  };
  const validLevels: JlptLevel[] = ['N5', 'N4', 'N3', 'N2', 'N1'];
  const validCategories: FlashcardCategory[] = ['vocabulary', 'kanji', 'grammar'];

  let settings: StoredStudySettings = { ...defaultSettings };
  let dataset: PlannedFlashcard[] = [];
  let validation: DatasetValidation = {
    status: 'idle',
    message: 'Chưa chọn file JSON. Dataset hiện tại sẽ được đọc từ chrome.storage.local nếu có.',
    validCount: 0,
    invalidCount: 0
  };
  let isLoading = true;
  let isSaving = false;
  let saveStatus = '';

  $: selectedLevelDescriptions = settings.selectedLevels.filter(isJlptLevel);

  onMount(() => {
    void loadOptions();
  });

  function hasChromeStorage() {
    return typeof chrome !== 'undefined' && Boolean(chrome.storage?.local);
  }

  function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
  }

  function isJlptLevel(value: unknown): value is JlptLevel {
    return typeof value === 'string' && validLevels.includes(value.toUpperCase() as JlptLevel);
  }

  function normalizeLevel(value: unknown): JlptLevel {
    return String(value).toUpperCase() as JlptLevel;
  }

  function isFlashcardCategory(value: unknown): value is FlashcardCategory {
    return typeof value === 'string' && validCategories.includes(value as FlashcardCategory);
  }

  function normalizeString(value: unknown) {
    return typeof value === 'string' ? value.trim() : '';
  }

  function normalizeNullableString(value: unknown) {
    if (value === null || value === undefined) return undefined;

    const normalized = normalizeString(value);
    return normalized.length > 0 ? normalized : undefined;
  }

  function readStorage(keys: Array<keyof StorageSnapshot>): Promise<StorageSnapshot> {
    if (!hasChromeStorage()) return Promise.resolve({});

    return chrome.storage.local.get(keys) as Promise<StorageSnapshot>;
  }

  function writeStorage(items: Record<string, unknown>) {
    if (!hasChromeStorage()) {
      saveStatus = 'Không tìm thấy chrome.storage.local trong môi trường hiện tại; thay đổi chỉ hiển thị trong phiên này.';
      return Promise.resolve();
    }

    return chrome.storage.local.set(items);
  }

  async function loadOptions() {
    const stored = await readStorage(storageKeys);
    const storedSettings = stored.jlptSettings ?? {};

    settings = {
      ...defaultSettings,
      ...storedSettings,
      dailyGoal: Number(storedSettings.dailyGoal ?? defaultSettings.dailyGoal),
      selectedLevels: normalizeStoredLevels(storedSettings.selectedLevels),
      enabledCategories: normalizeStoredCategories(storedSettings.enabledCategories),
      notificationIntervalMinutes: clampNumber(
        storedSettings.notificationIntervalMinutes,
        defaultSettings.notificationIntervalMinutes,
        1,
        1440
      ),
      notificationDisplaySeconds: clampNumber(
        storedSettings.notificationDisplaySeconds,
        defaultSettings.notificationDisplaySeconds,
        5,
        120
      ),
      notificationStartTime: normalizeTime(storedSettings.notificationStartTime),
      orderMode: storedSettings.orderMode === 'random' ? 'random' : 'sequential',
      theme: normalizeTheme(storedSettings.theme),
      notificationEnabled: Boolean(storedSettings.notificationEnabled)
    };

    dataset = stored.jlptDataset ?? stored.dataset ?? [];
    validation = {
      status: dataset.length > 0 ? 'valid' : 'idle',
      message:
        dataset.length > 0
          ? `Đã tải ${dataset.length} card từ chrome.storage.local.`
          : 'Chưa có dataset trong chrome.storage.local. Hãy chọn file JSON để nạp dữ liệu.',
      validCount: dataset.length,
      invalidCount: 0
    };
    isLoading = false;
  }

  function normalizeStoredLevels(levels: unknown) {
    if (!Array.isArray(levels)) return defaultSettings.selectedLevels;

    const normalized = levels.map(normalizeLevel).filter(isJlptLevel);
    return normalized.length > 0 ? Array.from(new Set(normalized)) : defaultSettings.selectedLevels;
  }

  function normalizeStoredCategories(categories: unknown) {
    if (!Array.isArray(categories)) return defaultSettings.enabledCategories;

    const normalized = categories.filter(isFlashcardCategory);
    return normalized.length > 0 ? Array.from(new Set(normalized)) : defaultSettings.enabledCategories;
  }

  function clampNumber(value: unknown, fallback: number, min: number, max: number) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;

    return Math.min(max, Math.max(min, Math.round(parsed)));
  }

  function normalizeTime(value: unknown) {
    if (typeof value !== 'string' || !/^\d{2}:\d{2}$/.test(value)) return defaultSettings.notificationStartTime;

    return value;
  }

  function normalizeTheme(value: unknown): ThemeMode {
    return value === 'light' || value === 'dark' || value === 'system' ? value : 'system';
  }

  function validateCard(rawCard: unknown, index: number): PlannedFlashcard | undefined {
    if (!isRecord(rawCard)) return undefined;

    const level = normalizeLevel(rawCard.level);
    const category = rawCard.category;
    const prompt = normalizeString(rawCard.prompt ?? rawCard.name);
    const answer = normalizeString(rawCard.answer ?? rawCard.mean);

    if (!isJlptLevel(level) || !isFlashcardCategory(category) || prompt.length === 0 || answer.length === 0) {
      return undefined;
    }

    return {
      id: normalizeNullableString(rawCard.id) ?? `${level}-${category}-${index + 1}`,
      level,
      category,
      prompt,
      answer,
      name: normalizeNullableString(rawCard.name) ?? prompt,
      mean: normalizeNullableString(rawCard.mean) ?? answer,
      reading: normalizeNullableString(rawCard.reading ?? rawCard.hiragana),
      hiragana: normalizeNullableString(rawCard.hiragana ?? rawCard.reading),
      example: normalizeNullableString(rawCard.example) ?? null,
      notes: normalizeNullableString(rawCard.notes),
      image: normalizeNullableString(rawCard.image) ?? null,
      audio: normalizeNullableString(rawCard.audio) ?? null
    };
  }

  function validateDataset(rawDataset: unknown) {
    if (!Array.isArray(rawDataset)) {
      return {
        validCards: [],
        validation: {
          status: 'invalid' as const,
          message: 'JSON phải là một array các flashcard.',
          validCount: 0,
          invalidCount: 0
        }
      };
    }

    const validCards = rawDataset
      .map((card, index) => validateCard(card, index))
      .filter((card): card is PlannedFlashcard => Boolean(card));
    const invalidCount = rawDataset.length - validCards.length;

    return {
      validCards,
      validation: {
        status: validCards.length === 0 ? ('invalid' as const) : invalidCount > 0 ? ('partial' as const) : ('valid' as const),
        message:
          validCards.length === 0
            ? 'Không tìm thấy card hợp lệ. Mỗi card cần level, category, prompt/name và answer/mean.'
            : invalidCount > 0
              ? `Đã lưu ${validCards.length} card hợp lệ và bỏ qua ${invalidCount} card lỗi.`
              : `Dataset hợp lệ. Đã lưu ${validCards.length} card.`,
        validCount: validCards.length,
        invalidCount
      }
    };
  }

  function readFileAsText(file: File) {
    return new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result ?? ''));
      reader.onerror = () => reject(new Error('Không thể đọc file JSON.'));
      reader.readAsText(file);
    });
  }

  async function handleDatasetFile(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    saveStatus = '';

    try {
      const text = await readFileAsText(file);
      const parsed = JSON.parse(text) as unknown;
      const result = validateDataset(parsed);

      validation = { ...result.validation, fileName: file.name };

      if (result.validCards.length === 0) return;

      dataset = result.validCards;
      await persistDataset(dataset);
    } catch (error) {
      validation = {
        status: 'invalid',
        message: `Lỗi parse JSON: ${error instanceof Error ? error.message : 'File không hợp lệ.'}`,
        validCount: 0,
        invalidCount: 0,
        fileName: file.name
      };
    } finally {
      input.value = '';
    }
  }

  async function persistDataset(cards: PlannedFlashcard[]) {
    isSaving = true;
    await writeStorage({ jlptDataset: cards, jlptCurrentIndex: 0 });
    isSaving = false;
    saveStatus = hasChromeStorage() ? 'Đã lưu dataset vào chrome.storage.local.' : saveStatus;
  }

  async function persistSettings() {
    settings = {
      ...settings,
      dailyGoal: clampNumber(settings.dailyGoal, defaultSettings.dailyGoal, 1, 999),
      notificationIntervalMinutes: clampNumber(settings.notificationIntervalMinutes, 60, 1, 1440),
      notificationDisplaySeconds: clampNumber(settings.notificationDisplaySeconds, 20, 5, 120)
    };

    isSaving = true;
    await writeStorage({ jlptSettings: settings, jlptNotificationPaused: !settings.notificationEnabled });
    isSaving = false;
    saveStatus = hasChromeStorage() ? 'Đã lưu settings vào chrome.storage.local.' : saveStatus;
  }

  function updateLevels(event: CustomEvent<JlptLevel[]>) {
    settings = { ...settings, selectedLevels: event.detail };
    void persistSettings();
  }

  function updateSetting<K extends keyof StoredStudySettings>(key: K, value: StoredStudySettings[K]) {
    settings = { ...settings, [key]: value };
    void persistSettings();
  }

  async function resetDataset() {
    dataset = [];
    validation = {
      status: 'idle',
      message: 'Đã reset dataset. Popup sẽ yêu cầu nạp file JSON mới.',
      validCount: 0,
      invalidCount: 0
    };
    isSaving = true;
    await writeStorage({ jlptDataset: [], jlptCurrentIndex: 0 });
    isSaving = false;
    saveStatus = hasChromeStorage() ? 'Đã xóa dataset trong chrome.storage.local.' : saveStatus;
  }

  async function resetSettings() {
    settings = { ...defaultSettings };
    isSaving = true;
    await writeStorage({ jlptSettings: settings, jlptNotificationPaused: !settings.notificationEnabled });
    isSaving = false;
    saveStatus = hasChromeStorage() ? 'Đã reset settings trong chrome.storage.local.' : saveStatus;
  }
</script>

<main class="app-shell options-shell">
  <section class="hero">
    <h1>Study settings</h1>
    <p>Tune your extension defaults for daily JLPT practice. All options are saved in chrome.storage.local.</p>
  </section>

  {#if isLoading}
    <section class="panel empty-state">
      <h2>Đang tải cài đặt…</h2>
      <p>Vui lòng chờ trong giây lát.</p>
    </section>
  {:else}
    <section class="panel settings-grid">
      <div class="section-heading">
        <div>
          <h2>Dataset</h2>
          <p>Nạp file JSON flashcard. Card hợp lệ cần level, category, prompt/name và answer/mean.</p>
        </div>
        <button class="secondary-button compact-button" type="button" on:click={resetDataset}>Reset dataset</button>
      </div>

      <label class="setting-row">
        <span>JSON file</span>
        <input accept="application/json,.json" type="file" on:change={handleDatasetFile} />
      </label>

      <div class:status-card={true} class:status-card--error={validation.status === 'invalid'}>
        <strong>{validation.fileName ?? 'Dataset status'}</strong>
        <p>{validation.message}</p>
        <p>{validation.validCount} card hợp lệ · {validation.invalidCount} card lỗi · {dataset.length} card đang lưu</p>
      </div>
    </section>

    <section class="panel settings-grid">
      <div class="section-heading">
        <h2>Study mode</h2>
      </div>

      <label class="radio-row">
        <input
          checked={settings.orderMode === 'random'}
          name="study-mode"
          type="radio"
          value="random"
          on:change={() => updateSetting('orderMode', 'random')}
        />
        <span>Random</span>
      </label>

      <label class="radio-row">
        <input
          checked={settings.orderMode === 'sequential'}
          name="study-mode"
          type="radio"
          value="sequential"
          on:change={() => updateSetting('orderMode', 'sequential')}
        />
        <span>Sequential</span>
      </label>
    </section>

    <section class="panel settings-grid">
      <div class="section-heading">
        <h2>Notifications</h2>
      </div>

      <label class="switch-row">
        <span>Enable reminder notifications</span>
        <input
          checked={settings.notificationEnabled}
          role="switch"
          type="checkbox"
          on:change={(event) => updateSetting('notificationEnabled', event.currentTarget.checked)}
        />
      </label>

      <label class="setting-row">
        <span>Interval minutes</span>
        <input
          min="1"
          max="1440"
          type="number"
          bind:value={settings.notificationIntervalMinutes}
          on:change={persistSettings}
        />
      </label>

      <label class="setting-row">
        <span>Reminder window starts at</span>
        <input type="time" bind:value={settings.notificationStartTime} on:change={persistSettings} />
      </label>

      <label class="setting-row">
        <span>Display duration hint (seconds)</span>
        <input
          min="5"
          max="120"
          type="number"
          bind:value={settings.notificationDisplaySeconds}
          on:change={persistSettings}
        />
      </label>

      <p class="help-text">
        Chrome Notifications API không đảm bảo thời lượng hiển thị cố định; hệ điều hành có thể tự quyết định.
        MVP lưu giá trị này để service worker có thể dùng làm hint hoặc tự clear notification khi phù hợp.
      </p>
    </section>

    <section class="panel settings-grid">
      <div class="section-heading">
        <h2>Appearance</h2>
      </div>

      <label class="setting-row">
        <span>Theme</span>
        <select bind:value={settings.theme} on:change={persistSettings}>
          <option value="light">Light</option>
          <option value="dark">Dark</option>
          <option value="system">System</option>
        </select>
      </label>
    </section>

    <section class="panel settings-grid">
      <div class="section-heading">
        <div>
          <h2>General settings</h2>
          <p>Daily goal and active JLPT levels are also persisted to chrome.storage.local.</p>
        </div>
        <button class="secondary-button compact-button" type="button" on:click={resetSettings}>Reset settings</button>
      </div>

      <label class="setting-row">
        <span>Daily review goal</span>
        <input bind:value={settings.dailyGoal} min="1" type="number" on:change={persistSettings} />
      </label>

      <div class="setting-row">
        <span>Active JLPT levels</span>
        <LevelSelector selectedLevels={settings.selectedLevels} on:change={updateLevels} />
      </div>
    </section>

    <section class="panel settings-grid">
      <h2>Level guide</h2>
      {#each selectedLevelDescriptions as level}
        <article>
          <strong>{level}</strong>
          <p>{levelDescriptions[level]}</p>
        </article>
      {/each}
    </section>

    {#if saveStatus || isSaving}
      <p class="save-status">{isSaving ? 'Đang lưu…' : saveStatus}</p>
    {/if}
  {/if}
</main>
