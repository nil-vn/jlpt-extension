<script lang="ts">
  import { Bell, BookOpenCheck, CheckCircle2, Database, FileJson, Info, ListOrdered, Moon, RotateCcw, Settings2, Shuffle, SlidersHorizontal, Sun, Target, UploadCloud } from '@lucide/svelte';
  import { onMount } from 'svelte';
  import '../app.css';
  import { LevelSelector } from '../lib/components';
  import { Badge } from '../lib/components/ui/badge';
  import { Button } from '../lib/components/ui/button';
  import { Card } from '../lib/components/ui/card';
  import { levelDescriptions } from '../lib/data';
  import { validateDataset, type DatasetValidationError } from '../lib/data/dataset-validator';
  import {
    DEFAULT_SETTINGS,
    NOTIFICATION_INTERVAL_OPTIONS,
    getExtensionState,
    hasChromeStorage,
    saveDataset,
    updateSettings,
    normalizeNotificationInterval,
    type UserSettings
  } from '../lib/extension/storage';
  import type { Flashcard, FlashcardCategory, JlptLevel } from '../lib/types/flashcard';

  type StoredStudySettings = UserSettings;
  type DatasetValidation = {
    status: 'idle' | 'valid' | 'partial' | 'invalid';
    message: string;
    validCount: number;
    invalidCount: number;
    errors: DatasetValidationError[];
    fileName?: string;
  };

  const defaultSettings: StoredStudySettings = { ...DEFAULT_SETTINGS };
  const validLevels: JlptLevel[] = ['n5', 'n4', 'n3', 'n2', 'n1'];
  const validCategories: FlashcardCategory[] = ['gramma', 'locabulary', 'kanji', 'reading', 'listening'];

  let settings: StoredStudySettings = { ...defaultSettings };
  let dataset: Flashcard[] = [];
  let validation: DatasetValidation = {
    status: 'idle',
    message: 'Chưa chọn file JSON. Dataset hiện tại sẽ được đọc từ chrome.storage.local nếu có.',
    validCount: 0,
    invalidCount: 0,
    errors: []
  };
  let isLoading = true;
  let isSaving = false;
  let saveStatus = '';

  $: selectedLevelDescriptions = settings.selectedLevels.filter(isJlptLevel);

  onMount(() => {
    void loadOptions();
  });

  function isJlptLevel(value: unknown): value is JlptLevel {
    return typeof value === 'string' && validLevels.includes(value.toLowerCase() as JlptLevel);
  }

  function normalizeLevel(value: unknown): JlptLevel {
    return String(value).toLowerCase() as JlptLevel;
  }

  function normalizeCategory(value: unknown): FlashcardCategory {
    const normalized = String(value).toLowerCase();

    if (normalized === 'grammar') return 'gramma';
    if (normalized === 'vocabulary') return 'locabulary';

    return normalized as FlashcardCategory;
  }

  function isFlashcardCategory(value: unknown): value is FlashcardCategory {
    return typeof value === 'string' && validCategories.includes(normalizeCategory(value));
  }

  async function loadOptions() {
    const stored = await getExtensionState();

    settings = {
      ...defaultSettings,
      ...stored.settings,
      dailyGoal: Number(stored.settings.dailyGoal ?? defaultSettings.dailyGoal),
      selectedLevels: normalizeStoredLevels(stored.settings.selectedLevels),
      enabledCategories: normalizeStoredCategories(stored.settings.enabledCategories),
      notificationIntervalMinutes: normalizeNotificationInterval(stored.settings.notificationIntervalMinutes),
      orderMode: stored.settings.orderMode === 'random' ? 'random' : 'sequential',
      theme: normalizeTheme(stored.settings.theme),
      notificationEnabled: Boolean(stored.settings.notificationEnabled)
    };

    dataset = stored.dataset;
    validation = {
      status: dataset.length > 0 ? 'valid' : 'idle',
      message:
        dataset.length > 0
          ? `Đã tải ${dataset.length} card từ chrome.storage.local.`
          : 'Chưa có dataset trong chrome.storage.local. Hãy chọn file JSON để nạp dữ liệu.',
      validCount: dataset.length,
      invalidCount: 0,
      errors: []
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

    const normalized = categories.map(normalizeCategory).filter(isFlashcardCategory);
    return normalized.length > 0 ? Array.from(new Set(normalized)) : defaultSettings.enabledCategories;
  }

  function clampNumber(value: unknown, fallback: number, min: number, max: number) {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return fallback;

    return Math.min(max, Math.max(min, Math.round(parsed)));
  }

  function normalizeTheme(value: unknown): UserSettings['theme'] {
    return value === 'dark' ? 'dark' : 'light';
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
      const hasErrors = result.errors.length > 0;

      validation = {
        status: hasErrors ? 'invalid' : 'valid',
        message: hasErrors
          ? `Dataset có ${result.errors.length} lỗi. Vui lòng sửa file JSON trước khi lưu.`
          : `Dataset hợp lệ. Đã lưu ${result.validCards.length} card.`,
        validCount: result.validCards.length,
        invalidCount: result.errors.length,
        errors: result.errors,
        fileName: file.name
      };

      if (hasErrors) return;

      dataset = result.validCards;
      await persistDataset(dataset);
    } catch (error) {
      validation = {
        status: 'invalid',
        message: `Lỗi parse JSON: ${error instanceof Error ? error.message : 'File không hợp lệ.'}`,
        validCount: 0,
        invalidCount: 1,
        errors: [{ message: error instanceof Error ? error.message : 'File không hợp lệ.' }],
        fileName: file.name
      };
    } finally {
      input.value = '';
    }
  }

  async function persistDataset(cards: Flashcard[]) {
    isSaving = true;
    await saveDataset(cards);
    isSaving = false;
    saveStatus = hasChromeStorage()
      ? 'Đã lưu dataset vào chrome.storage.local.'
      : 'Không tìm thấy chrome.storage.local trong môi trường hiện tại; dataset chỉ được lưu tạm trong phiên dev.';
  }

  async function persistSettings() {
    settings = {
      ...settings,
      dailyGoal: clampNumber(settings.dailyGoal, defaultSettings.dailyGoal, 1, 999),
      notificationIntervalMinutes: normalizeNotificationInterval(settings.notificationIntervalMinutes)
    };

    isSaving = true;
    await updateSettings(settings);
    isSaving = false;
    saveStatus = hasChromeStorage()
      ? 'Đã lưu settings vào chrome.storage.local.'
      : 'Không tìm thấy chrome.storage.local trong môi trường hiện tại; settings chỉ được lưu tạm trong phiên dev.';
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
      invalidCount: 0,
      errors: []
    };
    isSaving = true;
    await saveDataset([]);
    isSaving = false;
    saveStatus = hasChromeStorage()
      ? 'Đã xóa dataset trong chrome.storage.local.'
      : 'Không tìm thấy chrome.storage.local trong môi trường hiện tại; dataset chỉ được reset tạm trong phiên dev.';
  }

  async function resetSettings() {
    settings = { ...defaultSettings };
    isSaving = true;
    await updateSettings(settings);
    isSaving = false;
    saveStatus = hasChromeStorage()
      ? 'Đã reset settings trong chrome.storage.local.'
      : 'Không tìm thấy chrome.storage.local trong môi trường hiện tại; settings chỉ được reset tạm trong phiên dev.';
  }
</script>

<main class="app-shell options-shell">
  <section class="hero options-hero">
    <Badge class="hero__eyebrow" variant="outline"><Settings2 size={14} /> Control center</Badge>
    <h1>Study settings</h1>
    <p>Tune your extension defaults for daily JLPT practice. All options are saved in chrome.storage.local.</p>
  </section>

  {#if isLoading}
    <Card class="empty-state">
      <h2>Đang tải cài đặt…</h2>
      <p>Vui lòng chờ trong giây lát.</p>
    </Card>
  {:else}
    <Card class="settings-grid">
      <div class="section-heading">
        <div>
          <h2><Database size={20} /> Dataset</h2>
          <p>Nạp file JSON flashcard. Card chỉ được lưu khi đủ field và không có lỗi validation.</p>
        </div>
        <Button class="compact-button" variant="outline" on:click={resetDataset}><RotateCcw size={16} /> Reset dataset</Button>
      </div>

      <div class="setting-row">
        <span><FileJson size={16} /> JSON file</span>
        <label class="file-dropzone">
          <UploadCloud size={24} />
          <span>Chọn hoặc kéo thả file JSON flashcard</span>
          <input accept="application/json,.json" type="file" on:change={handleDatasetFile} />
        </label>
      </div>

      <div class:status-card={true} class:status-card--error={validation.status === 'invalid'}>
        <strong><CheckCircle2 size={16} /> {validation.fileName ?? 'Dataset status'}</strong>
        <p>{validation.message}</p>
        <p>{validation.validCount} card hợp lệ · {validation.invalidCount} lỗi validation · {dataset.length} card đang lưu</p>
        {#if validation.errors.length > 0}
          <ul class="validation-errors" aria-label="Dataset validation errors">
            {#each validation.errors.slice(0, 5) as error}
              <li>{error.message}</li>
            {/each}
          </ul>
          {#if validation.errors.length > 5}
            <p>…và {validation.errors.length - 5} lỗi khác.</p>
          {/if}
        {/if}
      </div>
    </Card>

    <Card class="settings-grid settings-grid--compact">
      <div class="section-heading">
        <h2><Shuffle size={20} /> Study mode</h2>
      </div>

      <div class="segmented-toggle" role="group" aria-label="Study mode">
        <button
          class:active={settings.orderMode === 'random'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.orderMode === 'random'}
          on:click={() => updateSetting('orderMode', 'random')}
        >
          <Shuffle size={14} />
          <span>Random</span>
        </button>
        <button
          class:active={settings.orderMode === 'sequential'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.orderMode === 'sequential'}
          on:click={() => updateSetting('orderMode', 'sequential')}
        >
          <ListOrdered size={14} />
          <span>Sequential</span>
        </button>
      </div>
    </Card>

    <Card class="settings-grid">
      <div class="section-heading">
        <h2><Bell size={20} /> Notifications</h2>
      </div>

      <label class="ios-switch-row">
        <span>Enable reminder notifications</span>
        <input
          checked={settings.notificationEnabled}
          role="switch"
          type="checkbox"
          on:change={(event) => updateSetting('notificationEnabled', event.currentTarget.checked)}
        />
        <span class="ios-switch-track" aria-hidden="true">
          <span class="ios-switch-icon ios-switch-icon--off">Off</span>
          <span class="ios-switch-icon ios-switch-icon--on">On</span>
          <span class="ios-switch-thumb"></span>
        </span>
      </label>

      <label class="setting-row">
        <span>Interval Display</span>
        <select
          disabled={!settings.notificationEnabled}
          value={settings.notificationIntervalMinutes}
          on:change={(event) => updateSetting('notificationIntervalMinutes', Number(event.currentTarget.value))}
        >
          {#each NOTIFICATION_INTERVAL_OPTIONS as option}
            <option value={option.minutes}>{option.label}</option>
          {/each}
        </select>
      </label>

      <p class="help-text">
        Bật notification để service worker tạo alarm theo interval đã cấu hình; thay đổi interval sẽ tự cập nhật alarm đang chạy.
        Thời lượng hiển thị notification do Chrome và hệ điều hành tự xử lý.
      </p>
    </Card>

    <Card class="settings-grid settings-grid--compact">
      <div class="section-heading">
        <h2><Sun size={20} /> Appearance</h2>
      </div>

      <div class="segmented-toggle" role="group" aria-label="Theme">
        <button
          class:active={settings.theme === 'light'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.theme === 'light'}
          on:click={() => updateSetting('theme', 'light')}
        >
          <Sun size={14} />
          <span>Light</span>
        </button>
        <button
          class:active={settings.theme === 'dark'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.theme === 'dark'}
          on:click={() => updateSetting('theme', 'dark')}
        >
          <Moon size={14} />
          <span>Dark</span>
        </button>
      </div>
    </Card>

    <Card class="settings-grid">
      <div class="section-heading">
        <div>
          <h2><SlidersHorizontal size={20} /> General settings</h2>
          <p>Daily goal and active JLPT levels are also persisted to chrome.storage.local.</p>
        </div>
        <Button class="compact-button" variant="outline" on:click={resetSettings}><RotateCcw size={16} /> Reset settings</Button>
      </div>

      <label class="setting-row">
        <span><Target size={16} /> Daily review goal</span>
        <input bind:value={settings.dailyGoal} min="1" type="number" on:change={persistSettings} />
      </label>

      <div class="setting-row">
        <span>Active JLPT levels</span>
        <LevelSelector selectedLevels={settings.selectedLevels} on:change={updateLevels} />
      </div>
    </Card>

    <Card class="settings-grid level-guide">
      <h2><BookOpenCheck size={20} /> Level guide</h2>
      {#each selectedLevelDescriptions as level}
        <article>
          <strong><Info size={16} /> {level.toUpperCase()}</strong>
          <p>{levelDescriptions[level]}</p>
        </article>
      {/each}
    </Card>

    {#if saveStatus || isSaving}
      <p class="save-status">{isSaving ? 'Đang lưu…' : saveStatus}</p>
    {/if}
  {/if}
</main>
