<script lang="ts">
  import { Bookmark, BookmarkCheck, ChevronLeft, ChevronRight, Eye, EyeOff, Settings, Sparkles } from '@lucide/svelte';
  import { onDestroy, onMount } from 'svelte';
  import { LevelSelector, StudyCard } from '../../lib/components';
  import { Badge } from '../../lib/components/ui/badge';
  import { Button } from '../../lib/components/ui/button';
  import { Card } from '../../lib/components/ui/card';
  import { starterFlashcards } from '../../lib/data';
  import { translate, type AppLanguage } from '../../lib/i18n';
  import {
    DEFAULT_SETTINGS,
    getExtensionState,
    hasChromeStorage,
    saveNote,
    toggleBookmark as toggleStoredBookmark,
    updateCurrentIndex,
    updateSettings,
    type UserSettings
  } from '../../lib/extension/storage';
  import { createFlashcardId, type Flashcard, type JlptLevel } from '../../lib/types/flashcard';

  type StudyMode = UserSettings['orderMode'];
  type PlannedFlashcard = Flashcard & {
    level: string;
    category: string;
  };

  export let openSettings: () => void = () => {};

  const NOTE_SAVE_DELAY_MS = 400;
  const defaultSelectedLevels: JlptLevel[] = ['n5'];

  let dataset: PlannedFlashcard[] = [];
  let selectedLevels: JlptLevel[] = defaultSelectedLevels;
  let enabledCategories: PlannedFlashcard['category'][] = [];
  let currentIndex = 0;
  let revealed = false;
  let studyMode: StudyMode = 'sequential';
  let storedSettings: UserSettings = { ...DEFAULT_SETTINGS };
  let bookmarkedCardIds: string[] = [];
  let notesByCardId: Record<string, string> = {};
  let currentNote = '';
  let noteSaveTimer: ReturnType<typeof setTimeout> | undefined;
  let historyStack: number[] = [];
  let showBookmarkedOnly = false;
  let isLoading = true;
  let language: AppLanguage = 'en';

  $: cardsById = new Map(dataset.map((card) => [getCardId(card), card]));
  $: levelFilteredCards = dataset.filter(
    (card) => selectedLevels.includes(normalizeLevel(card.level)) && enabledCategories.includes(card.category)
  );
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
  $: t = (key: Parameters<typeof translate>[1], params?: Parameters<typeof translate>[2]) => translate(language, key, params);
  $: bookmarkReviewButtonLabel = showBookmarkedOnly ? t('returnToLevelCards') : t('reviewBookmarked');
  $: studyProgressLabel = showBookmarkedOnly
    ? t('bookmarksProgress', { current: studyCards.length === 0 ? 0 : currentIndex + 1, total: studyCards.length })
    : t('cardProgress', { current: studyCards.length === 0 ? 0 : currentIndex + 1, total: studyCards.length });

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
    applyStoredState(stored);
    isLoading = false;
  }

  function applyStoredState(stored: Awaited<ReturnType<typeof getExtensionState>>) {
    dataset = stored.dataset.length > 0 ? stored.dataset : hasChromeStorage() ? [] : starterFlashcards;
    selectedLevels = stored.settings.selectedLevels.map(normalizeLevel).filter((level, index, levels) => levels.indexOf(level) === index);
    enabledCategories = stored.settings.enabledCategories;
    storedSettings = stored.settings;
    studyMode = stored.settings.orderMode;
    currentIndex = stored.currentIndex;
    bookmarkedCardIds = stored.bookmarkedCardIds;
    notesByCardId = stored.notesByCardId;
    revealed = stored.settings.revealAnswers;
    language = stored.settings.language;
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
    storedSettings = { ...storedSettings, selectedLevels, orderMode: studyMode };
    void updateSettings({ selectedLevels, orderMode: studyMode }).then(() => updateCurrentIndex(currentIndex));
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
</script>

<main class="app-shell popup-shell desktop-study-view">
  <section class="hero">
    <div class="hero__header">
      <div>
        <Badge class="hero__eyebrow" variant="outline"><Sparkles size={14} /> Desktop Study</Badge>
        <h1>JLPT Study Companion</h1>
        <p>{t('jlptDescription')}</p>
      </div>
      <Button class="compact-button hero__settings" variant="secondary" on:click={openSettings}><Settings size={16} /> {t('settings')}</Button>
    </div>
  </section>

  <Card class="level-panel">
    <div class="panel-label">{t('chooseLevel')}</div>
    <LevelSelector {selectedLevels} on:change={updateLevels} />
  </Card>

  {#if isLoading}
    <Card class="empty-state">
      <h2>{t('loadingData')}</h2>
      <p>{t('validLevelOnly')}</p>
    </Card>
  {:else if dataset.length === 0}
    <Card class="empty-state">
      <h2>{t('emptyDatasetTitle')}</h2>
      <p>{t('emptyDatasetDescription')}</p>
      <Button on:click={openSettings}><Settings size={16} /> {t('openSettings')}</Button>
    </Card>
  {:else if currentCard}
    <StudyCard card={currentCard} {revealed} {language}>
      <div class="study-card-controls" slot="header-actions">
        <Button class="study-card-bookmark-button" variant={isBookmarked ? 'secondary' : 'outline'} size="sm" on:click={toggleBookmark} aria-pressed={isBookmarked}>
          {#if isBookmarked}
            <BookmarkCheck size={16} />
            {t('bookmarkAdded')}
          {:else}
            <Bookmark size={16} />
            {t('bookmarkReview')}
          {/if}
        </Button>
        <Badge class="study-progress-badge" variant={showBookmarkedOnly ? 'success' : 'secondary'}>{studyProgressLabel}</Badge>
      </div>

      <label class="note-field study-card__note" slot="footer">
        <span>{t('noteLabel')}</span>
        <textarea
          bind:value={currentNote}
          on:input={scheduleNoteSave}
          rows="4"
          placeholder={t('notePlaceholder')}
        ></textarea>
      </label>
    </StudyCard>

    <div class="actions learning-pagination">
      <Button variant="outline" on:click={previousCard}><ChevronLeft size={16} /> {t('previous')}</Button>
      <Button on:click={toggleRevealed} aria-pressed={revealed}>
        {#if revealed}
          <EyeOff size={16} />
          {t('hideAnswer')}
        {:else}
          <Eye size={16} />
          {t('showAnswer')}
        {/if}
      </Button>
      <Button variant="outline" on:click={nextCard}>{t('next')} <ChevronRight size={16} /></Button>
    </div>
  {:else}
    <Card class="empty-state">
      <h2>{t('noMatchingCardsTitle')}</h2>
      <p>{showBookmarkedOnly ? t('bookmarkEmpty') : t('noMatchingCardsDescription')}</p>
    </Card>
  {/if}

  {#if !isLoading}
    <Card class="bookmark-panel">
      <div class="bookmark-panel__header">
        <div>
          <div class="panel-label"><BookmarkCheck size={16} /> {t('bookmarkPanelTitle')}</div>
          <p class="help-text">{t('bookmarkHelp', { count: bookmarkedCards.length })}</p>
        </div>
        <Button variant={showBookmarkedOnly ? 'secondary' : 'outline'} on:click={toggleBookmarkReview} disabled={bookmarkedCards.length === 0}>
          <Bookmark size={16} />
          {bookmarkReviewButtonLabel}
        </Button>
      </div>

      {#if bookmarkedCards.length > 0}
        <div class="bookmark-list" aria-label={t('bookmarkListLabel')}>
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
          <p class="help-text">{t('bookmarkLatestHelp')}</p>
        {/if}
      {/if}
    </Card>
  {/if}
</main>
