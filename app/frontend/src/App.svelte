<script lang="ts">
  import { Events } from '@wailsio/runtime';
  import { AppService } from '../bindings/github.com/nil-vn/jlpt-extension/app';
  import type {
    AppStatus,
    FlashcardDTO,
    ImportResult,
    LibrarySummary,
    ValidationError,
  } from '../bindings/github.com/nil-vn/jlpt-extension/app/models';

  type SelectedImportFile = {
    name: string;
    size: number;
    content: string;
    itemCount: number | null;
    parseError: string | null;
  };

  let status = $state<AppStatus | null>(null);
  let time = $state('Đang chờ event từ Wails...');
  let selectedFile = $state<SelectedImportFile | null>(null);
  let previewResult = $state<ImportResult | null>(null);
  let importResult = $state<ImportResult | null>(null);
  let library = $state<FlashcardDTO[]>([]);
  let summary = $state<LibrarySummary | null>(null);
  let search = $state('');
  let isBusy = $state(false);
  let isDragging = $state(false);
  let errorMessage = $state<string | null>(null);

  let canImport = $derived(selectedFile !== null && !isBusy);
  let latestResult = $derived(importResult ?? previewResult);
  let hasValidationErrors = $derived((latestResult?.errors?.length ?? 0) > 0);

  const formatBytes = (size: number): string => {
    if (size < 1024) return `${size} B`;
    if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  };

  const validationLocation = (item: ValidationError): string => {
    const row = item.index === undefined || item.index === null ? 'File' : `Dòng ${item.index + 1}`;
    return item.field ? `${row} · ${item.field}` : row;
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

  const refreshAll = async (): Promise<void> => {
    await Promise.all([loadStatus(), loadLibrary()]);
  };

  const readSelectedFile = async (file: File): Promise<void> => {
    errorMessage = null;
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
    isBusy = true;
    errorMessage = null;
    try {
      previewResult = await AppService.PreviewImportJSON(selectedFile.name, selectedFile.content);
    } catch (error) {
      errorMessage = error instanceof Error ? error.message : String(error);
    } finally {
      isBusy = false;
    }
  };

  const importFile = async (): Promise<void> => {
    if (!selectedFile) return;
    isBusy = true;
    errorMessage = null;
    importResult = null;
    try {
      importResult = await AppService.ImportFlashcardsFromJSON(selectedFile.name, selectedFile.content, {
        replaceLibrary: false,
        dryRun: false,
      });
      if ((importResult.errors?.length ?? 0) === 0) {
        await refreshAll();
      }
    } catch (error) {
      errorMessage = error instanceof Error ? error.message : String(error);
    } finally {
      isBusy = false;
    }
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

  refreshAll().catch((error: unknown) => {
    errorMessage = error instanceof Error ? error.message : String(error);
  });

  Events.On('time', (timeValue: { data: string }) => {
    time = timeValue.data;
  });
</script>

<main class="shell">
  <section class="hero">
    <div>
      <p class="eyebrow">Milestone 3 · Desktop import UI</p>
      <h1>Import JLPT JSON</h1>
      <p class="lede">
        Chọn hoặc kéo thả dataset JSON từ máy. Go backend sẽ validate schema, import vào SQLite và Library sẽ refresh ngay sau khi import thành công.
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
        <button type="button" class="secondary" disabled={!selectedFile || isBusy} onclick={previewImport}>
          Validate lại
        </button>
        <button type="button" disabled={!canImport} onclick={importFile}>
          {isBusy ? 'Đang xử lý...' : 'Import vào Library'}
        </button>
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
          <p class="success-message">Import thành công. Library bên dưới đã được refresh từ SQLite.</p>
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
              <span>{card.category}</span>
            </div>
            <h3>{card.name}</h3>
            <p class="hiragana">{card.hiragana}</p>
            <p>{card.mean}</p>
            {#if card.example}
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
