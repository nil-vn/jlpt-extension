<script lang="ts">
  import { Events } from '@wailsio/runtime';
  import { AppService } from '../bindings/github.com/nil-vn/jlpt-extension/app';
  import type {
    AppStatus,
    FlashcardDTO,
    ImportResult,
    LibrarySummary,
    StudySettingsDTO,
    StudyStateDTO,
    ValidationError,
    NotificationPayload,
  } from '../bindings/github.com/nil-vn/jlpt-extension/app/models';
  import LevelSelector from './lib/components/LevelSelector.svelte';
  import StudyCard from './lib/components/StudyCard.svelte';

  type SelectedImportFile = {
    name: string;
    size: number;
    content: string;
    itemCount: number | null;
    parseError: string | null;
  };

  const notificationIntervals = [
    { label: '30 giây', minutes: 0.5 },
    { label: '1 phút', minutes: 1 },
    { label: '3 phút', minutes: 3 },
    { label: '10 phút', minutes: 10 },
  ];

  const categories = [
    { value: 'gramma', label: 'Grammar' },
    { value: 'vocabulary', label: 'Vocabulary' },
    { value: 'kanji', label: 'Kanji' },
    { value: 'reading', label: 'Reading' },
    { value: 'listening', label: 'Listening' },
  ];

  let status = $state<AppStatus | null>(null);
  let time = $state('Đang chờ event từ Wails...');
  let selectedFile = $state<SelectedImportFile | null>(null);
  let previewResult = $state<ImportResult | null>(null);
  let importResult = $state<ImportResult | null>(null);
  let library = $state<FlashcardDTO[]>([]);
  let summary = $state<LibrarySummary | null>(null);
  let study = $state<StudyStateDTO | null>(null);
  let search = $state('');
  let isBusy = $state(false);
  let isDragging = $state(false);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);
  let lastNotification = $state<NotificationPayload | null>(null);

  let canImport = $derived(selectedFile !== null && !isBusy);
  let latestResult = $derived(importResult ?? previewResult);
  let hasValidationErrors = $derived((latestResult?.errors?.length ?? 0) > 0);
  let settings = $derived(study?.settings ?? null);

  const formatBytes = (size: number): string => {
    if (size < 1024) return `${size} B`;
    if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  };

  const validationLocation = (item: ValidationError): string => {
    const row = item.index === undefined || item.index === null ? 'File' : `Dòng ${item.index + 1}`;
    return item.field ? `${row} · ${item.field}` : row;
  };

  const runAction = async (action: () => Promise<void>): Promise<void> => {
    isBusy = true;
    errorMessage = null;
    successMessage = null;
    try {
      await action();
    } catch (error) {
      errorMessage = error instanceof Error ? error.message : String(error);
    } finally {
      isBusy = false;
    }
  };

  const loadStatus = async (): Promise<void> => {
    status = await AppService.Status();
  };

  const loadLibrary = async (): Promise<void> => {
    const [cards, librarySummary] = await Promise.all([
      AppService.ListFlashcards({ level: '', category: '', search, limit: 24, offset: 0 }),
      AppService.LibrarySummary(),
    ]);
    library = cards;
    summary = librarySummary;
  };

  const loadStudy = async (): Promise<void> => {
    study = await AppService.GetStudyState();
  };

  const refreshAll = async (): Promise<void> => {
    await Promise.all([loadStatus(), loadLibrary(), loadStudy()]);
  };

  const readSelectedFile = async (file: File): Promise<void> => {
    errorMessage = null;
    successMessage = null;
    previewResult = null;
    importResult = null;

    if (!file.name.toLowerCase().endsWith('.json')) {
      errorMessage = 'Vui lòng chọn file có phần mở rộng .json.';
      selectedFile = null;
      return;
    }

    const content = await file.text();
    let itemCount: number | null = null;
    let parseError: string | null = null;
    try {
      const parsed = JSON.parse(content);
      if (Array.isArray(parsed)) {
        itemCount = parsed.length;
      } else {
        parseError = 'Root JSON phải là array flashcard.';
      }
    } catch (error) {
      parseError = error instanceof Error ? error.message : String(error);
    }

    selectedFile = { name: file.name, size: file.size, content, itemCount, parseError };
    await previewImport();
  };

  const previewImport = async (): Promise<void> => {
    if (!selectedFile) return;
    await runAction(async () => {
      previewResult = await AppService.PreviewImportJSON(selectedFile!.name, selectedFile!.content);
    });
  };

  const importFile = async (): Promise<void> => {
    if (!selectedFile) return;
    await runAction(async () => {
      importResult = await AppService.ImportFlashcardsFromJSON(selectedFile!.name, selectedFile!.content, {
        replaceLibrary: false,
        dryRun: false,
      });
      if ((importResult.errors?.length ?? 0) === 0) {
        await refreshAll();
        successMessage = 'Import thành công. Study, settings và Library đã refresh từ SQLite.';
      }
    });
  };

  const handleInputChange = async (event: Event): Promise<void> => {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (file) await readSelectedFile(file);
    input.value = '';
  };

  const handleDrop = async (event: DragEvent): Promise<void> => {
    event.preventDefault();
    isDragging = false;
    const file = event.dataTransfer?.files?.[0];
    if (file) await readSelectedFile(file);
  };

  const handleDragOver = (event: DragEvent): void => {
    event.preventDefault();
    isDragging = true;
  };

  const handleDragLeave = (): void => {
    isDragging = false;
  };

  const handleSearch = async (): Promise<void> => {
    await loadLibrary();
  };

  const moveNext = async (): Promise<void> => {
    await runAction(async () => {
      study = await AppService.MoveNext();
    });
  };

  const movePrevious = async (): Promise<void> => {
    await runAction(async () => {
      study = await AppService.MovePrevious();
    });
  };

  const updateSettings = async (patch: Partial<StudySettingsDTO>): Promise<void> => {
    if (!study) return;
    await runAction(async () => {
      study = await AppService.UpdateStudySettings({ ...study!.settings, ...patch });
    });
  };

  const toggleCategory = async (category: string): Promise<void> => {
    if (!settings) return;
    const enabledCategories = settings.enabledCategories.includes(category)
      ? settings.enabledCategories.filter((item) => item !== category)
      : [...settings.enabledCategories, category];
    await updateSettings({ enabledCategories });
  };

  const toggleNotifications = async (enabled: boolean): Promise<void> => {
    await updateSettings({ notificationEnabled: enabled, notificationPaused: !enabled });
  };

  const setNotificationPaused = async (paused: boolean): Promise<void> => {
    await runAction(async () => {
      study = await AppService.SetNotificationPaused(paused);
      successMessage = paused ? 'Đã tạm dừng notification.' : 'Đã bật lại notification scheduler.';
    });
  };

  const sendTestNotification = async (): Promise<void> => {
    await runAction(async () => {
      const payload = await AppService.ShowStudyNotificationNow();
      await showDesktopNotification(payload);
      await refreshAll();
      successMessage = 'Đã gửi notification test từ Go scheduler.';
    });
  };

  const showDesktopNotification = async (payload: NotificationPayload): Promise<void> => {
    lastNotification = payload;
    if (!('Notification' in window)) return;

    let permission = Notification.permission;
    if (permission === 'default') {
      permission = await Notification.requestPermission();
    }
    if (permission !== 'granted') return;

    new Notification(payload.title, {
      body: payload.message,
      tag: payload.id,
    });
  };

  const saveNote = async (note: string): Promise<void> => {
    const cardID = study?.currentCard?.id;
    if (!cardID) return;
    await runAction(async () => {
      const card = await AppService.SaveNote(cardID, note);
      study = { ...study!, currentCard: card };
      successMessage = 'Đã lưu note vào SQLite.';
    });
  };

  const toggleBookmark = async (): Promise<void> => {
    const cardID = study?.currentCard?.id;
    if (!cardID) return;
    await runAction(async () => {
      const card = await AppService.ToggleBookmark(cardID);
      study = { ...study!, currentCard: card };
      await loadLibrary();
      successMessage = card.bookmarked ? 'Đã bookmark card.' : 'Đã bỏ bookmark card.';
    });
  };

  const revealCurrent = async (): Promise<void> => {
    await updateSettings({ revealAnswers: true });
  };

  refreshAll().catch((error: unknown) => {
    errorMessage = error instanceof Error ? error.message : String(error);
  });

  Events.On('time', (timeValue: { data: string }) => {
    time = timeValue.data;
  });

  Events.On('flashcard-notification', (event: { data: NotificationPayload }) => {
    void showDesktopNotification(event.data);
  });
</script>

<main class="shell">
  <section class="hero">
    <div>
      <p class="eyebrow">Milestone 5 · Notifications desktop</p>
      <h1>JLPT Desktop Study</h1>
      <p class="lede">
        Import dataset JSON, học flashcard từ SQLite và để Go scheduler phát notification định kỳ khi app đang mở.
      </p>
    </div>
    <div class="hero-stat" aria-label="Library total">
      <span>{status?.libraryCount ?? summary?.total ?? 0}</span>
      <small>cards trong Library</small>
    </div>
  </section>

  {#if errorMessage}
    <section class="alert error" role="alert">{errorMessage}</section>
  {/if}
  {#if successMessage}
    <section class="alert success" role="status">{successMessage}</section>
  {/if}

  <section class="workspace study-workspace">
    <article class="card study-panel">
      <div class="section-heading">
        <div>
          <p class="eyebrow muted">Study</p>
          <h2>Flashcard hiện tại</h2>
        </div>
        <span class="pill">{(study?.currentIndex ?? -1) + 1}/{study?.totalCards ?? 0}</span>
      </div>

      <StudyCard
        card={study?.currentCard ?? null}
        revealAnswers={study?.settings.revealAnswers ?? false}
        onreveal={revealCurrent}
        ontogglebookmark={toggleBookmark}
        onsavenote={saveNote}
      />

      <div class="actions study-actions">
        <button type="button" class="secondary" disabled={isBusy || !study?.currentCard} onclick={movePrevious}>Previous</button>
        <button type="button" disabled={isBusy || !study?.currentCard} onclick={moveNext}>Next</button>
      </div>
    </article>

    <article class="card settings-panel">
      <div class="section-heading">
        <div>
          <p class="eyebrow muted">Settings</p>
          <h2>Filter & mode</h2>
        </div>
        <span class="pill">SQLite</span>
      </div>

      {#if settings}
        <div class="settings-stack">
          <label>
            <span>Study mode</span>
            <div class="segmented">
              <button type="button" class:active={settings.orderMode === 'sequential'} onclick={() => updateSettings({ orderMode: 'sequential' })}>Sequential</button>
              <button type="button" class:active={settings.orderMode === 'random'} onclick={() => updateSettings({ orderMode: 'random' })}>Random</button>
            </div>
          </label>

          <label>
            <span>JLPT levels</span>
            <LevelSelector selectedLevels={settings.selectedLevels} onchange={(selectedLevels) => updateSettings({ selectedLevels })} />
          </label>

          <label>
            <span>Categories</span>
            <div class="category-grid">
              {#each categories as category}
                <button
                  type="button"
                  class="chip"
                  class:active={settings.enabledCategories.includes(category.value)}
                  aria-pressed={settings.enabledCategories.includes(category.value)}
                  onclick={() => toggleCategory(category.value)}
                >
                  {category.label}
                </button>
              {/each}
            </div>
          </label>

          <label>
            <span>Daily goal</span>
            <input
              type="number"
              min="1"
              value={settings.dailyGoal}
              onchange={(event) => updateSettings({ dailyGoal: Number((event.currentTarget as HTMLInputElement).value) })}
            />
          </label>

          <div class="notification-settings">
            <div class="notification-row">
              <div>
                <strong>Notifications</strong>
                <small>{settings.notificationEnabled && !settings.notificationPaused ? `Đang bật mỗi ${settings.notificationIntervalMinutes} phút` : 'Đang tắt hoặc tạm dừng'}</small>
              </div>
              <label class="switch">
                <input
                  type="checkbox"
                  checked={settings.notificationEnabled && !settings.notificationPaused}
                  onchange={(event) => toggleNotifications(event.currentTarget.checked)}
                />
                <span>Enable</span>
              </label>
            </div>

            <label>
              <span>Notification interval</span>
              <select
                value={settings.notificationIntervalMinutes}
                disabled={!settings.notificationEnabled}
                onchange={(event) => updateSettings({ notificationIntervalMinutes: Number((event.currentTarget as HTMLSelectElement).value) })}
              >
                {#each notificationIntervals as option}
                  <option value={option.minutes}>{option.label}</option>
                {/each}
              </select>
            </label>

            <div class="actions compact-actions">
              <button type="button" class="secondary" disabled={!settings.notificationEnabled || settings.notificationPaused || isBusy} onclick={() => setNotificationPaused(true)}>Pause</button>
              <button type="button" class="secondary" disabled={!settings.notificationEnabled || !settings.notificationPaused || isBusy} onclick={() => setNotificationPaused(false)}>Resume</button>
              <button type="button" disabled={!settings.notificationEnabled || settings.notificationPaused || isBusy || !study?.currentCard} onclick={sendTestNotification}>Test now</button>
            </div>

            {#if lastNotification}
              <p class="notification-preview"><strong>{lastNotification.title}</strong><br />{lastNotification.message}</p>
            {/if}
          </div>
        </div>
      {:else}
        <p class="empty-state">Đang tải settings từ SQLite...</p>
      {/if}
    </article>
  </section>

  <section class="workspace">
    <article class="card import-card">
      <div class="section-heading">
        <div>
          <p class="eyebrow muted">Import JSON</p>
          <h2>Chọn file từ máy</h2>
        </div>
        <span class="pill">Max 25 MB</span>
      </div>

      <label
        class:dragging={isDragging}
        class="dropzone"
        for="json-file"
        ondragover={handleDragOver}
        ondragleave={handleDragLeave}
        ondrop={handleDrop}
      >
        <input id="json-file" type="file" accept="application/json,.json" onchange={handleInputChange} />
        <span class="drop-icon">⬆</span>
        <strong>Kéo thả JSON vào đây</strong>
        <small>hoặc bấm để mở file picker của hệ điều hành/WebView</small>
      </label>

      {#if selectedFile}
        <div class="file-meta">
          <div>
            <strong>{selectedFile.name}</strong>
            <span>{formatBytes(selectedFile.size)}</span>
          </div>
          <div>
            <strong>{selectedFile.itemCount ?? '—'}</strong>
            <span>items đọc được</span>
          </div>
        </div>
        {#if selectedFile.parseError}
          <p class="inline-warning">Preview JSON: {selectedFile.parseError}</p>
        {/if}
      {/if}

      <div class="actions">
        <button type="button" class="secondary" disabled={!selectedFile || isBusy} onclick={previewImport}>Validate lại</button>
        <button type="button" disabled={!canImport} onclick={importFile}>{isBusy ? 'Đang xử lý...' : 'Import vào Library'}</button>
      </div>
    </article>

    <article class="card result-card">
      <div class="section-heading">
        <div>
          <p class="eyebrow muted">Validation / Import result</p>
          <h2>Kết quả</h2>
        </div>
        {#if latestResult}
          <span class:success={!hasValidationErrors} class:error-pill={hasValidationErrors} class="pill">
            {hasValidationErrors ? 'Có lỗi' : importResult ? 'Imported' : 'Valid'}
          </span>
        {/if}
      </div>

      {#if latestResult}
        <div class="metrics">
          <div><strong>{latestResult.totalRows}</strong><span>Total</span></div>
          <div><strong>{latestResult.validRows}</strong><span>Valid</span></div>
          <div><strong>{latestResult.invalidRows}</strong><span>Invalid</span></div>
          <div><strong>{latestResult.inserted}</strong><span>Inserted</span></div>
          <div><strong>{latestResult.updated}</strong><span>Updated</span></div>
        </div>

        {#if latestResult.sourceSha256}
          <p class="hash">SHA-256: {latestResult.sourceSha256}</p>
        {/if}

        {#if hasValidationErrors}
          <ul class="errors">
            {#each latestResult.errors.slice(0, 8) as item}
              <li><strong>{validationLocation(item)}</strong><span>{item.message}</span></li>
            {/each}
          </ul>
          {#if latestResult.errors.length > 8}
            <p class="inline-warning">Còn {latestResult.errors.length - 8} lỗi khác. Sửa file JSON rồi validate lại.</p>
          {/if}
        {:else if importResult}
          <p class="success-message">Import thành công. Library và Study đã được refresh từ SQLite.</p>
        {:else}
          <p class="success-message">File hợp lệ. Bấm “Import vào Library” để ghi vào SQLite.</p>
        {/if}
      {:else}
        <p class="empty-state">Chưa có file. Hãy chọn hoặc kéo thả JSON để xem kết quả validation.</p>
      {/if}
    </article>
  </section>

  <section class="card library-card">
    <div class="section-heading">
      <div>
        <p class="eyebrow muted">Library</p>
        <h2>Flashcards đã import</h2>
      </div>
      <div class="search-row">
        <input aria-label="Tìm flashcard" placeholder="Tìm theo name, mean, hiragana" bind:value={search} oninput={handleSearch} />
        <button type="button" class="secondary" onclick={loadLibrary}>Refresh</button>
      </div>
    </div>

    {#if summary}
      <div class="level-strip" aria-label="Cards by JLPT level">
        {#each Object.entries(summary.byLevel) as [level, count]}
          <span>{level.toUpperCase()} · {count}</span>
        {/each}
      </div>
    {/if}

    {#if library.length > 0}
      <div class="library-grid">
        {#each library as card}
          <article class="flashcard">
            <div class="flashcard-top">
              <span>{card.level.toUpperCase()}</span>
              <span>{card.bookmarked ? '★' : ''} {card.category}</span>
            </div>
            <h3>{card.name}</h3>
            <p class="hiragana">{card.hiragana}</p>
            <p>{card.mean}</p>
            {#if card.note}
              <small>Note: {card.note}</small>
            {:else if card.example}
              <small>{card.example}</small>
            {/if}
          </article>
        {/each}
      </div>
    {:else}
      <p class="empty-state">Library đang trống. Import một file JSON hợp lệ để thấy card trong app.</p>
    {/if}
  </section>

  <footer>{time}</footer>
</main>
