<script lang="ts">
  import { Bell, Bookmark, BookmarkCheck, ChevronLeft, ChevronRight, Eye, EyeOff, Settings, Sparkles } from '@lucide/svelte';
  import { onDestroy, onMount } from 'svelte';
  import '../app.css';
  import { LevelSelector, StudyCard } from '../lib/components';
  import { Badge } from '../lib/components/ui/badge';
  import { Button } from '../lib/components/ui/button';
  import { Card } from '../lib/components/ui/card';
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
  let notificationFeedback = '';
  let noteSaveTimer: ReturnType<typeof setTimeout> | undefined;
  let historyStack: number[] = [];
  let showBookmarkedOnly = false;
  let isLoading = true;

  $: cardsById = new Map(dataset.map((card) => [getCardId(card), card]));
  $: levelFilteredCards = dataset.filter((card) => selectedLevels.includes(normalizeLevel(card.level)));
  $: bookmarkedCards = bookmarkedCardIds.map((cardId) => cardsById.get(cardId)).filter(isPlannedFlashcard).reverse();
  $: studyCards = showBookmarkedOnly ? bookmarkedCards : levelFilteredCards;
  $: if (studyCards.length > 0 && currentIndex >= studyCards.length) {
    currentIndex = 0;
  }
  $: if (showBookmarkedOnly && bookmarkedCards.length === 0) {
    showBookmarkedOnly = false;
  }
  $: currentCard = studyCards.length > 0 ? studyCards[currentIndex] : undefined;
  $: currentCardId = currentCard ? getCardId(currentCard) : '';
  $: isBookmarked = currentCardId ? bookmarkedCardIds.includes(currentCardId) : false;
  $: currentNote = currentCardId ? notesByCardId[currentCardId] ?? '' : '';
  $: bookmarkReviewButtonLabel = showBookmarkedOnly ? 'Quay lại thẻ theo cấp độ' : 'Mở lại từ đã bookmark';
  $: studyProgressLabel = showBookmarkedOnly
    ? `Bookmark ${studyCards.length === 0 ? 0 : currentIndex + 1}/${studyCards.length}`
    : `Thẻ ${studyCards.length === 0 ? 0 : currentIndex + 1}/${studyCards.length}`;
  $: notificationButtonLabel = notificationPaused ? 'Resume notification' : 'Pause notification';
  $: notificationStatusLabel = notificationPaused
    ? 'Notifications đang tạm dừng cho tới khi bạn bật lại.'
    : `Notifications đang bật mỗi ${formatNotificationInterval(storedSettings?.notificationIntervalMinutes)}.`;

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

  function isPlannedFlashcard(card: PlannedFlashcard | undefined): card is PlannedFlashcard {
    return Boolean(card);
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
    revealed = stored.settings.revealAnswers;
    isLoading = false;
  }

  function nextCard() {
    if (studyCards.length === 0) return;

    historyStack = [...historyStack, currentIndex];
    currentIndex = studyMode === 'random' ? nextRandomIndex() : (currentIndex + 1) % studyCards.length;
    void updateCurrentIndex(currentIndex);
  }

  function previousCard() {
    if (studyCards.length === 0) return;

    if (studyMode === 'random' && historyStack.length > 0) {
      currentIndex = historyStack[historyStack.length - 1];
      historyStack = historyStack.slice(0, -1);
    } else {
      currentIndex = (currentIndex - 1 + studyCards.length) % studyCards.length;
    }
    void updateCurrentIndex(currentIndex);
  }

  function nextRandomIndex() {
    if (studyCards.length < 2) return currentIndex;

    let nextIndex = currentIndex;
    while (nextIndex === currentIndex) {
      nextIndex = Math.floor(Math.random() * studyCards.length);
    }

    return nextIndex;
  }

  function updateLevels(event: CustomEvent<JlptLevel[]>) {
    selectedLevels = event.detail;
    currentIndex = 0;
    historyStack = [];
    storedSettings = { ...storedSettings, selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused };
    void updateSettings({ selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused }).then(() =>
      updateCurrentIndex(currentIndex)
    );
  }

  function toggleRevealed() {
    revealed = !revealed;
    storedSettings = { ...storedSettings, revealAnswers: revealed };
    void updateSettings({ revealAnswers: revealed }).then((state) => {
      storedSettings = state.settings;
      revealed = state.settings.revealAnswers;
    });
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


  function toggleBookmarkReview() {
    if (!showBookmarkedOnly && bookmarkedCards.length === 0) return;

    showBookmarkedOnly = !showBookmarkedOnly;
    currentIndex = 0;
    historyStack = [];
    void updateCurrentIndex(currentIndex);
  }

  function openBookmarkedCard(cardId: string) {
    const bookmarkedIndex = bookmarkedCards.findIndex((card) => getCardId(card) === cardId);
    if (bookmarkedIndex === -1) return;

    showBookmarkedOnly = true;
    currentIndex = bookmarkedIndex;
    historyStack = [];
    void updateCurrentIndex(currentIndex);
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
    const nextPaused = !notificationPaused;
    notificationPaused = nextPaused;
    storedSettings = { ...storedSettings, selectedLevels, orderMode: studyMode, notificationEnabled: !notificationPaused };
    void setNotificationPaused(notificationPaused).then((state) => {
      storedSettings = state.settings;
      notificationPaused = state.notificationPaused ?? !state.settings.notificationEnabled;

      if (!notificationPaused) {
        void showNotificationNow().then((result) => {
          notificationFeedback = result.ok
            ? 'Đã gửi notification test. Nếu vẫn không thấy, hãy kiểm tra quyền thông báo của Chrome trong hệ điều hành.'
            : `Không gửi được notification: ${result.error ?? 'Lỗi không xác định.'}`;
        });
      }
    });
  }

  function showNotificationNow(): Promise<{ ok: boolean; error?: string }> {
    if (typeof chrome === 'undefined' || !chrome.runtime?.sendMessage) {
      return Promise.resolve({ ok: false, error: 'Chrome runtime API không khả dụng.' });
    }

    return chrome.runtime.sendMessage({ type: 'SHOW_STUDY_NOTIFICATION_NOW' });
  }

  function formatNotificationInterval(intervalMinutes: number | undefined) {
    if (intervalMinutes === 0.5) return '30 giây';
    if (intervalMinutes === 1) return '1 phút';

    return `${intervalMinutes ?? 1} phút`;
  }

  function openOptionsPage() {
    if (typeof chrome !== 'undefined' && chrome.runtime?.openOptionsPage) {
      chrome.runtime.openOptionsPage();
      return;
    }

    window.open('options.html', '_blank', 'noopener');
  }
</script>

<main class="app-shell popup-shell">
  <section class="hero">
    <div class="hero__header">
      <div>
        <Badge class="hero__eyebrow" variant="outline"><Sparkles size={14} /> Daily JLPT</Badge>
        <h1>JLPT Study Companion</h1>
        <p>Ôn flashcard JLPT, nghe audio, ghi chú và đánh dấu xem lại.</p>
      </div>
      <Button class="compact-button hero__settings" variant="secondary" on:click={openOptionsPage}><Settings size={16} /> Cài đặt</Button>
    </div>
  </section>

  <Card class="level-panel">
    <div class="panel-label">Chọn cấp độ</div>
    <LevelSelector {selectedLevels} on:change={updateLevels} />
  </Card>

  <Card class="bookmark-panel">
    <div class="bookmark-panel__header">
      <div>
        <div class="panel-label"><BookmarkCheck size={16} /> Từ vựng đã bookmark</div>
        <p class="help-text">{bookmarkedCards.length} từ đã lưu để mở lại và ôn nhanh.</p>
      </div>
      <Button variant={showBookmarkedOnly ? 'secondary' : 'outline'} on:click={toggleBookmarkReview} disabled={bookmarkedCards.length === 0}>
        <Bookmark size={16} />
        {bookmarkReviewButtonLabel}
      </Button>
    </div>

    {#if bookmarkedCards.length > 0}
      <div class="bookmark-list" aria-label="Danh sách từ vựng đã bookmark">
        {#each bookmarkedCards.slice(0, 5) as bookmarkedCard (getCardId(bookmarkedCard))}
          <button
            class:active={showBookmarkedOnly && currentCardId === getCardId(bookmarkedCard)}
            class="bookmark-chip"
            type="button"
            on:click={() => openBookmarkedCard(getCardId(bookmarkedCard))}
          >
            <span>{bookmarkedCard.name}</span>
            <small>{bookmarkedCard.hiragana || bookmarkedCard.mean}</small>
          </button>
        {/each}
      </div>
      {#if bookmarkedCards.length > 5}
        <p class="help-text">Hiển thị 5 từ bookmark mới nhất. Bấm “Mở lại từ đã bookmark” để duyệt tất cả.</p>
      {/if}
    {/if}
  </Card>

  {#if isLoading}
    <Card class="empty-state">
      <h2>Đang tải dữ liệu…</h2>
      <p>Vui lòng chờ trong giây lát.</p>
    </Card>
  {:else if dataset.length === 0}
    <Card class="empty-state">
      <h2>Chưa có dataset</h2>
      <p>Hãy mở trang cài đặt và nạp file JSON flashcard trước khi bắt đầu học.</p>
      <Button on:click={openOptionsPage}><Settings size={16} /> Mở trang cài đặt</Button>
    </Card>
  {:else if currentCard}
    <div class="study-progress" aria-live="polite">
      <Badge variant={showBookmarkedOnly ? 'success' : 'secondary'}>{studyProgressLabel}</Badge>
    </div>

    <StudyCard card={currentCard} {revealed} />

    <div class="actions learning-pagination">
      <Button variant="outline" on:click={previousCard}><ChevronLeft size={16} /> Trước</Button>
      <Button on:click={toggleRevealed} aria-pressed={revealed}>
        {#if revealed}
          <EyeOff size={16} />
          Ẩn đáp án
        {:else}
          <Eye size={16} />
          Hiện đáp án
        {/if}
      </Button>
      <Button variant="outline" on:click={nextCard}>Sau <ChevronRight size={16} /></Button>
    </div>

    <Card class="study-tools">
      <div class="actions actions--wrap">
        <Button variant={isBookmarked ? 'secondary' : 'outline'} on:click={toggleBookmark} aria-pressed={isBookmarked}>
          {#if isBookmarked}
            <BookmarkCheck size={16} />
            Đã đánh dấu
          {:else}
            <Bookmark size={16} />
            Đánh dấu xem lại
          {/if}
        </Button>
        <label class="notification-toggle">
          <span><Bell size={16} /> {notificationButtonLabel}</span>
          <input
            checked={!notificationPaused}
            role="switch"
            type="checkbox"
            on:change={toggleNotifications}
          />
          <span class="ios-switch-track" aria-hidden="true">
            <span class="ios-switch-icon ios-switch-icon--off">Off</span>
            <span class="ios-switch-icon ios-switch-icon--on">On</span>
            <span class="ios-switch-thumb"></span>
          </span>
        </label>
      </div>

      <p class="help-text">{notificationStatusLabel}</p>
      {#if notificationFeedback}
        <p class="help-text">{notificationFeedback}</p>
      {/if}

      <label class="note-field">
        <span>Ghi chú cho thẻ hiện tại</span>
        <textarea
          bind:value={currentNote}
          on:input={scheduleNoteSave}
          rows="4"
          placeholder="Nhập mnemonic, ví dụ riêng hoặc điểm cần xem lại…"
        ></textarea>
      </label>
    </Card>
  {:else}
    <Card class="empty-state">
      <h2>Không có thẻ phù hợp</h2>
      <p>{showBookmarkedOnly ? 'Bạn chưa có bookmark nào để mở lại.' : 'Dataset đã được nạp, nhưng chưa có thẻ nào thuộc level đang chọn.'}</p>
    </Card>
  {/if}
</main>
