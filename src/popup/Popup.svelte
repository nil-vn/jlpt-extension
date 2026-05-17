<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import '../app.css';
  import { LevelSelector, StudyCard } from '../lib/components';
  import { starterFlashcards } from '../lib/data';
  import {
    getExtensionState,
    hasChromeStorage,
    saveNote,
    setNotificationPaused,
    toggleBookmark as toggleStoredBookmark,
    updateCurrentIndex,
    updateSettings,
    type UserSettings
  } from '../lib/extension/storage';
  import { createFlashcardId, type Flashcard, type JlptLevel } from '../lib/types/flashcard';

  type StudyMode = UserSettings['orderMode'];
  type PlannedFlashcard = Flashcard & {
    level: string;
    category: string;
  };

  const NOTE_SAVE_DELAY_MS = 400;
  const defaultSelectedLevels: JlptLevel[] = ['n5'];

  let dataset: PlannedFlashcard[] = [];
  let selectedLevels: JlptLevel[] = defaultSelectedLevels;
  let currentIndex = 0;
  let revealed = false;
  let studyMode: StudyMode = 'sequential';
  let storedSettings: UserSettings;
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
  $: notificationButtonLabel = notificationPaused ? 'Resume notification' : 'Pause notification';
  $: notificationStatusLabel = notificationPaused
    ? 'Notifications đang tạm dừng cho tới khi bạn bật lại.'
    : `Notifications đang bật mỗi ${storedSettings?.notificationIntervalMinutes ?? 60} phút.`;

  onMount(() => {
    void loadState();
  });

  onDestroy(() => {
    if (noteSaveTimer) {
      clearTimeout(noteSaveTimer);
    }
  });

  function normalizeLevel(level: string): JlptLevel {
    return level.toLowerCase() as JlptLevel;
  }

  function getCardId(card: PlannedFlashcard) {
    return createFlashcardId(card);
  }

  async function loadState() {
    const stored = await getExtensionState();

    dataset = stored.dataset.length > 0 ? stored.dataset : hasChromeStorage() ? [] : starterFlashcards;
    selectedLevels = stored.settings.selectedLevels.map(normalizeLevel).filter((level, index, levels) => levels.indexOf(level) === index);
    storedSettings = stored.settings;
    studyMode = stored.settings.orderMode;
    currentIndex = stored.currentIndex;
    bookmarkedCardIds = stored.bookmarkedCardIds;
    notesByCardId = stored.notesByCardId;
    notificationPaused = stored.notificationPaused ?? !stored.settings.notificationEnabled;
    isLoading = false;
  }

  function nextCard() {
    if (filteredCards.length === 0) return;

    revealed = false;
    historyStack = [...historyStack, currentIndex];
    currentIndex = studyMode === 'random' ? nextRandomIndex() : (currentIndex + 1) % filteredCards.length;
    void updateCurrentIndex(currentIndex);
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
    void updateCurrentIndex(currentIndex);
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
    void updateSettings({ selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused }).then(() =>
      updateCurrentIndex(currentIndex)
    );
  }

  function toggleBookmark() {
    if (!currentCardId) return;

    const nextBookmarkedCardIds = isBookmarked
      ? bookmarkedCardIds.filter((cardId) => cardId !== currentCardId)
      : [...bookmarkedCardIds, currentCardId];
    bookmarkedCardIds = nextBookmarkedCardIds;
    void toggleStoredBookmark(currentCardId).then((state) => {
      bookmarkedCardIds = state.bookmarkedCardIds;
    });
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
      void saveNote(cardId, note);
    }, NOTE_SAVE_DELAY_MS);
  }

  function toggleNotifications() {
    notificationPaused = !notificationPaused;
    storedSettings = { ...storedSettings, selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused };
    void setNotificationPaused(notificationPaused).then((state) => {
      storedSettings = state.settings;
      notificationPaused = state.notificationPaused ?? !state.settings.notificationEnabled;
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
        <button class="secondary-button" type="button" on:click={toggleNotifications} aria-pressed={!notificationPaused}>
          {notificationButtonLabel}
        </button>
      </div>

      <p class="help-text">{notificationStatusLabel}</p>

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
