<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import '../app.css';
  import { LevelSelector, StudyCard } from '../lib/components';
  import { starterFlashcards } from '../lib/data';
  import type { Flashcard, JlptLevel } from '../lib/types/flashcard';

  type StudyMode = 'sequential' | 'random';
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
  type StoredStudySettings = {
    selectedLevels?: string[];
    orderMode?: StudyMode;
    notificationEnabled?: boolean;
  };
  type StorageSnapshot = {
    jlptDataset?: PlannedFlashcard[];
    dataset?: PlannedFlashcard[];
    jlptSettings?: StoredStudySettings;
    jlptCurrentIndex?: number;
    jlptBookmarkedCardIds?: string[];
    jlptNotesByCardId?: Record<string, string>;
    jlptNotificationPaused?: boolean;
  };

  const NOTE_SAVE_DELAY_MS = 400;
  const defaultSelectedLevels: JlptLevel[] = ['N5'];

  let dataset: PlannedFlashcard[] = [];
  let selectedLevels: JlptLevel[] = defaultSelectedLevels;
  let currentIndex = 0;
  let revealed = false;
  let studyMode: StudyMode = 'sequential';
  let storedSettings: StoredStudySettings = {};
  let bookmarkedCardIds: string[] = [];
  let notesByCardId: Record<string, string> = {};
  let currentNote = '';
  let notificationPaused = false;
  let noteSaveTimer: ReturnType<typeof setTimeout> | undefined;
  let historyStack: number[] = [];
  let isLoading = true;

  $: filteredCards = dataset.filter((card) => selectedLevels.includes(normalizeLevel(card.level)));
  $: if (filteredCards.length > 0 && currentIndex >= filteredCards.length) {
    currentIndex = 0;
  }
  $: currentCard = filteredCards.length > 0 ? filteredCards[currentIndex] : undefined;
  $: currentCardId = currentCard ? getCardId(currentCard) : '';
  $: isBookmarked = currentCardId ? bookmarkedCardIds.includes(currentCardId) : false;
  $: currentNote = currentCardId ? notesByCardId[currentCardId] ?? '' : '';

  onMount(() => {
    void loadState();
  });

  onDestroy(() => {
    if (noteSaveTimer) {
      clearTimeout(noteSaveTimer);
    }
  });

  function hasChromeStorage() {
    return typeof chrome !== 'undefined' && Boolean(chrome.storage?.local);
  }

  function normalizeLevel(level: string): JlptLevel {
    return level.toUpperCase() as JlptLevel;
  }

  function getCardId(card: PlannedFlashcard) {
    if (card.id) return card.id;

    return [card.level, card.category, card.name ?? card.prompt, card.hiragana ?? card.reading, card.mean ?? card.answer]
      .map((part) => String(part ?? '').trim())
      .join('|');
  }

  function readStorage(keys: Array<keyof StorageSnapshot>): Promise<StorageSnapshot> {
    if (!hasChromeStorage()) return Promise.resolve({});

    return chrome.storage.local.get(keys) as Promise<StorageSnapshot>;
  }

  function writeStorage(items: Record<string, unknown>) {
    if (!hasChromeStorage()) return Promise.resolve();

    return chrome.storage.local.set(items);
  }

  async function loadState() {
    const stored = await readStorage([
      'jlptDataset',
      'dataset',
      'jlptSettings',
      'jlptCurrentIndex',
      'jlptBookmarkedCardIds',
      'jlptNotesByCardId',
      'jlptNotificationPaused'
    ]);

    dataset = stored.jlptDataset ?? stored.dataset ?? (hasChromeStorage() ? [] : (starterFlashcards as PlannedFlashcard[]));
    selectedLevels = ((stored.jlptSettings?.selectedLevels ?? defaultSelectedLevels).map(normalizeLevel) as JlptLevel[]).filter(
      (level, index, levels) => levels.indexOf(level) === index
    );
    storedSettings = stored.jlptSettings ?? {};
    studyMode = storedSettings.orderMode ?? 'sequential';
    currentIndex = stored.jlptCurrentIndex ?? 0;
    bookmarkedCardIds = stored.jlptBookmarkedCardIds ?? [];
    notesByCardId = stored.jlptNotesByCardId ?? {};
    notificationPaused = stored.jlptNotificationPaused ?? stored.jlptSettings?.notificationEnabled === false;
    isLoading = false;
  }

  function nextCard() {
    if (filteredCards.length === 0) return;

    revealed = false;
    historyStack = [...historyStack, currentIndex];
    currentIndex = studyMode === 'random' ? nextRandomIndex() : (currentIndex + 1) % filteredCards.length;
    void writeStorage({ jlptCurrentIndex: currentIndex });
  }

  function previousCard() {
    if (filteredCards.length === 0) return;

    revealed = false;
    if (studyMode === 'random' && historyStack.length > 0) {
      currentIndex = historyStack[historyStack.length - 1];
      historyStack = historyStack.slice(0, -1);
    } else {
      currentIndex = (currentIndex - 1 + filteredCards.length) % filteredCards.length;
    }
    void writeStorage({ jlptCurrentIndex: currentIndex });
  }

  function nextRandomIndex() {
    if (filteredCards.length < 2) return currentIndex;

    let nextIndex = currentIndex;
    while (nextIndex === currentIndex) {
      nextIndex = Math.floor(Math.random() * filteredCards.length);
    }

    return nextIndex;
  }

  function updateLevels(event: CustomEvent<JlptLevel[]>) {
    selectedLevels = event.detail;
    currentIndex = 0;
    revealed = false;
    historyStack = [];
    storedSettings = { ...storedSettings, selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused };
    void writeStorage({
      jlptCurrentIndex: currentIndex,
      jlptSettings: {
        ...storedSettings,
        selectedLevels,
        orderMode: studyMode,
        notificationEnabled: !notificationPaused
      }
    });
  }

  function toggleBookmark() {
    if (!currentCardId) return;

    bookmarkedCardIds = isBookmarked
      ? bookmarkedCardIds.filter((cardId) => cardId !== currentCardId)
      : [...bookmarkedCardIds, currentCardId];
    void writeStorage({ jlptBookmarkedCardIds: bookmarkedCardIds });
  }

  function scheduleNoteSave() {
    if (!currentCardId) return;

    if (noteSaveTimer) {
      clearTimeout(noteSaveTimer);
    }

    const cardId = currentCardId;
    const note = currentNote;
    noteSaveTimer = setTimeout(() => {
      notesByCardId = { ...notesByCardId, [cardId]: note };
      void writeStorage({ jlptNotesByCardId: notesByCardId });
    }, NOTE_SAVE_DELAY_MS);
  }

  function toggleNotifications() {
    notificationPaused = !notificationPaused;
    storedSettings = { ...storedSettings, selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused };
    void writeStorage({
      jlptNotificationPaused: notificationPaused,
      jlptSettings: {
        ...storedSettings,
        selectedLevels,
        orderMode: studyMode,
        notificationEnabled: !notificationPaused
      }
    });
  }

  function openOptionsPage() {
    if (typeof chrome !== 'undefined' && chrome.runtime?.openOptionsPage) {
      chrome.runtime.openOptionsPage();
      return;
    }

    window.open('options.html', '_blank', 'noopener');
  }
</script>

<main class="app-shell">
  <section class="hero">
    <div class="hero__header">
      <div>
        <h1>JLPT Study Companion</h1>
        <p>Ôn flashcard JLPT, nghe audio, ghi chú và đánh dấu xem lại.</p>
      </div>
      <button class="secondary-button compact-button" type="button" on:click={openOptionsPage}>Cài đặt</button>
    </div>
  </section>

  <section class="panel">
    <LevelSelector {selectedLevels} on:change={updateLevels} />
  </section>

  {#if isLoading}
    <section class="panel empty-state">
      <h2>Đang tải dữ liệu…</h2>
      <p>Vui lòng chờ trong giây lát.</p>
    </section>
  {:else if dataset.length === 0}
    <section class="panel empty-state">
      <h2>Chưa có dataset</h2>
      <p>Hãy mở trang cài đặt và nạp file JSON flashcard trước khi bắt đầu học.</p>
      <button class="primary-button" type="button" on:click={openOptionsPage}>Mở trang cài đặt</button>
    </section>
  {:else if currentCard}
    <StudyCard card={currentCard} {revealed} />

    <div class="actions actions--wrap">
      <button class="secondary-button" type="button" on:click={previousCard}>Previous</button>
      <button class="primary-button" type="button" on:click={() => (revealed = !revealed)}>
        {revealed ? 'Ẩn đáp án' : 'Hiện đáp án'}
      </button>
      <button class="secondary-button" type="button" on:click={nextCard}>Next</button>
    </div>

    <section class="panel study-tools">
      <div class="actions actions--wrap">
        <button class="secondary-button" type="button" on:click={toggleBookmark} aria-pressed={isBookmarked}>
          {isBookmarked ? '★ Đã đánh dấu' : '☆ Đánh dấu xem lại'}
        </button>
        <button class="secondary-button" type="button" on:click={toggleNotifications} aria-pressed={notificationPaused}>
          {notificationPaused ? 'Resume notification' : 'Pause notification'}
        </button>
      </div>

      <label class="note-field">
        <span>Ghi chú cho thẻ hiện tại</span>
        <textarea
          bind:value={currentNote}
          on:input={scheduleNoteSave}
          rows="4"
          placeholder="Nhập mnemonic, ví dụ riêng hoặc điểm cần xem lại…"
        ></textarea>
      </label>
    </section>
  {:else}
    <section class="panel empty-state">
      <h2>Không có thẻ phù hợp</h2>
      <p>Dataset đã được nạp, nhưng chưa có thẻ nào thuộc level đang chọn.</p>
    </section>
  {/if}
</main>
