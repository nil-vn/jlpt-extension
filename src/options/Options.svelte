<script lang="ts">
  import { Bell, BookOpenCheck, CheckCircle2, Database, FileJson, Info, ListOrdered, Moon, RotateCcw, Settings2, Shuffle, SlidersHorizontal, Sun, Target, UploadCloud } from '@lucide/svelte';
  import { onMount } from 'svelte';
  import '../app.css';
  import { LevelSelector } from '../lib/components';
  import { Badge } from '../lib/components/ui/badge';
  import { Button } from '../lib/components/ui/button';
  import { Card } from '../lib/components/ui/card';
  import { formatInterval, supportedLanguages, translate, type AppLanguage, type TranslationKey } from '../lib/i18n';
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
  const validCategories: FlashcardCategory[] = ['gramma', 'vocabulary', 'kanji', 'reading', 'listening'];

  let settings: StoredStudySettings = { ...defaultSettings };
  let dataset: Flashcard[] = [];
  let validation: DatasetValidation = {
    status: 'idle',
    message: translate('en', 'datasetCurrentStatus'),
    validCount: 0,
    invalidCount: 0,
    errors: []
  };
  let isLoading = true;
  let isSaving = false;
  let saveStatus = '';

  $: language = settings.language;
  $: t = (key: TranslationKey, params?: Parameters<typeof translate>[2]) => translate(language, key, params);
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
    if (normalized === 'vocabulary') return 'vocabulary';

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
      notificationEnabled: Boolean(stored.settings.notificationEnabled),
      language: normalizeLanguage(stored.settings.language)
    };

    dataset = stored.dataset;
    validation = {
      status: dataset.length > 0 ? 'valid' : 'idle',
      message:
        dataset.length > 0
          ? t('datasetLoaded', { count: dataset.length })
          : t('datasetNoStored'),
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

  function normalizeLanguage(value: unknown): AppLanguage {
    return supportedLanguages.includes(value as AppLanguage) ? (value as AppLanguage) : 'en';
  }

  function getLevelDescriptionKey(level: JlptLevel): TranslationKey {
    return `levelDescription${level.toUpperCase()}` as TranslationKey;
  }

  function readFileAsText(file: File) {
    return new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result ?? ''));
      reader.onerror = () => reject(new Error(t('fileReadError')));
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
      const result = validateDataset(parsed, language);
      const hasErrors = result.errors.length > 0;

      validation = {
        status: hasErrors ? 'invalid' : 'valid',
        message: hasErrors
          ? t('datasetErrors', { count: result.errors.length })
          : t('datasetValid', { count: result.validCards.length }),
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
        message: t('parseJsonError', { message: error instanceof Error ? error.message : t('statusInvalidFile') }),
        validCount: 0,
        invalidCount: 1,
        errors: [{ message: error instanceof Error ? error.message : t('statusInvalidFile') }],
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
      ? t('chromeStorageDatasetSaved')
      : t('chromeStorageDatasetDev');
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
      ? t('chromeStorageSettingsSaved')
      : t('chromeStorageSettingsDev');
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
      message: t('datasetReset'),
      validCount: 0,
      invalidCount: 0,
      errors: []
    };
    isSaving = true;
    await saveDataset([]);
    isSaving = false;
    saveStatus = hasChromeStorage()
      ? t('chromeStorageDatasetReset')
      : t('chromeStorageDatasetResetDev');
  }

  async function resetSettings() {
    settings = { ...defaultSettings };
    isSaving = true;
    await updateSettings(settings);
    isSaving = false;
    saveStatus = hasChromeStorage()
      ? t('chromeStorageSettingsReset')
      : t('chromeStorageSettingsResetDev');
  }
</script>

<main class="app-shell options-shell">
  <section class="hero options-hero">
    <Badge class="hero__eyebrow" variant="outline"><Settings2 size={14} /> {t('controlCenter')}</Badge>
    <h1>{t('studySettings')}</h1>
    <p>{t('tuneDefaults')}</p>
  </section>

  {#if isLoading}
    <Card class="empty-state">
      <h2>{t('loadingSettings')}</h2>
      <p>{t('validLevelOnly')}</p>
    </Card>
  {:else}
    <Card class="settings-grid">
      <div class="section-heading">
        <div>
          <h2><Database size={20} /> {t('dataset')}</h2>
          <p>{t('datasetDescription')}</p>
        </div>
        <Button class="compact-button" variant="outline" on:click={resetDataset}><RotateCcw size={16} /> {t('resetDataset')}</Button>
      </div>

      <div class="setting-row">
        <span><FileJson size={16} /> {t('jsonFile')}</span>
        <label class="file-dropzone">
          <UploadCloud size={24} />
          <span>{t('chooseDatasetFile')}</span>
          <input accept="application/json,.json" type="file" on:change={handleDatasetFile} />
        </label>
      </div>

      <div class:status-card={true} class:status-card--error={validation.status === 'invalid'}>
        <strong><CheckCircle2 size={16} /> {validation.fileName ?? t('datasetStatus')}</strong>
        <p>{validation.message}</p>
        <p>{t('datasetSummary', { valid: validation.validCount, invalid: validation.invalidCount, stored: dataset.length })}</p>
        {#if validation.errors.length > 0}
          <ul class="validation-errors" aria-label={t('datasetValidationErrorsLabel')}>
            {#each validation.errors.slice(0, 5) as error}
              <li>{error.message}</li>
            {/each}
          </ul>
          {#if validation.errors.length > 5}
            <p>{t('otherErrors', { count: validation.errors.length - 5 })}</p>
          {/if}
        {/if}
      </div>
    </Card>

    <Card class="settings-grid settings-grid--compact">
      <div class="section-heading">
        <h2><Shuffle size={20} /> {t('studyMode')}</h2>
      </div>

      <div class="segmented-toggle" role="group" aria-label={t('studyMode')}>
        <button
          class:active={settings.orderMode === 'random'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.orderMode === 'random'}
          on:click={() => updateSetting('orderMode', 'random')}
        >
          <Shuffle size={14} />
          <span>{t('random')}</span>
        </button>
        <button
          class:active={settings.orderMode === 'sequential'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.orderMode === 'sequential'}
          on:click={() => updateSetting('orderMode', 'sequential')}
        >
          <ListOrdered size={14} />
          <span>{t('sequential')}</span>
        </button>
      </div>
    </Card>

    <Card class="settings-grid">
      <div class="section-heading">
        <h2><Bell size={20} /> {t('notifications')}</h2>
      </div>

      <label class="ios-switch-row">
        <span>{t('reminderNotifications')}</span>
        <input
          checked={settings.notificationEnabled}
          role="switch"
          type="checkbox"
          on:change={(event) => updateSetting('notificationEnabled', event.currentTarget.checked)}
        />
        <span class="ios-switch-track" aria-hidden="true">
          <span class="ios-switch-icon ios-switch-icon--off">{t('switchOff')}</span>
          <span class="ios-switch-icon ios-switch-icon--on">{t('switchOn')}</span>
          <span class="ios-switch-thumb"></span>
        </span>
      </label>

      <label class="setting-row">
        <span>{t('intervalDisplay')}</span>
        <select
          disabled={!settings.notificationEnabled}
          value={settings.notificationIntervalMinutes}
          on:change={(event) => updateSetting('notificationIntervalMinutes', Number(event.currentTarget.value))}
        >
          {#each NOTIFICATION_INTERVAL_OPTIONS as option}
            <option value={option.minutes}>{formatInterval(language, option.minutes)}</option>
          {/each}
        </select>
      </label>

      <p class="help-text">
        {t('notificationsHelp')}
      </p>
    </Card>

    <Card class="settings-grid settings-grid--compact">
      <div class="section-heading">
        <h2><Sun size={20} /> {t('appearance')}</h2>
      </div>

      <div class="segmented-toggle" role="group" aria-label={t('theme')}>
        <button
          class:active={settings.theme === 'light'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.theme === 'light'}
          on:click={() => updateSetting('theme', 'light')}
        >
          <Sun size={14} />
          <span>{t('light')}</span>
        </button>
        <button
          class:active={settings.theme === 'dark'}
          class="toggle-icon-button"
          type="button"
          aria-pressed={settings.theme === 'dark'}
          on:click={() => updateSetting('theme', 'dark')}
        >
          <Moon size={14} />
          <span>{t('dark')}</span>
        </button>
      </div>
    </Card>

    <Card class="settings-grid">
      <div class="section-heading">
        <div>
          <h2><SlidersHorizontal size={20} /> {t('generalSettings')}</h2>
          <p>{t('generalSettingsDescription')}</p>
        </div>
        <Button class="compact-button" variant="outline" on:click={resetSettings}><RotateCcw size={16} /> {t('resetSettings')}</Button>
      </div>

      <label class="setting-row">
        <span><Target size={16} /> {t('dailyGoal')}</span>
        <input bind:value={settings.dailyGoal} min="1" type="number" on:change={persistSettings} />
      </label>

      <label class="setting-row">
        <span>{t('language')}</span>
        <select
          value={settings.language}
          on:change={(event) => updateSetting('language', normalizeLanguage(event.currentTarget.value))}
        >
          <option value="en">{t('languageEnglish')}</option>
          <option value="vi">{t('languageVietnamese')}</option>
        </select>
      </label>

      <div class="setting-row">
        <span>{t('activeJlptLevels')}</span>
        <LevelSelector selectedLevels={settings.selectedLevels} on:change={updateLevels} />
      </div>
    </Card>

    <Card class="settings-grid level-guide">
      <h2><BookOpenCheck size={20} /> {t('levelGuide')}</h2>
      {#each selectedLevelDescriptions as level}
        <article>
          <strong><Info size={16} /> {level.toUpperCase()}</strong>
          <p>{t(getLevelDescriptionKey(level))}</p>
        </article>
      {/each}
    </Card>

    {#if saveStatus || isSaving}
      <p class="save-status">{isSaving ? t('loadingData') : saveStatus}</p>
    {/if}
  {/if}
</main>
