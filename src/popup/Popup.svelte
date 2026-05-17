<script lang="ts">
  import '../app.css';
  import { LevelSelector, StudyCard } from '../lib/components';
  import { starterFlashcards } from '../lib/data';
  import type { JlptLevel } from '../lib/types/flashcard';

  let selectedLevels: JlptLevel[] = ['N5'];
  let currentIndex = 0;
  let revealed = false;

  $: filteredCards = starterFlashcards.filter((card) => selectedLevels.includes(card.level));
  $: currentCard = filteredCards[currentIndex % Math.max(filteredCards.length, 1)];

  function nextCard() {
    revealed = false;
    currentIndex = filteredCards.length === 0 ? 0 : (currentIndex + 1) % filteredCards.length;
  }

  function updateLevels(event: CustomEvent<JlptLevel[]>) {
    selectedLevels = event.detail;
    currentIndex = 0;
    revealed = false;
  }
</script>

<main class="app-shell">
  <section class="hero">
    <h1>JLPT Study Companion</h1>
    <p>Review a focused deck before your next reading session.</p>
  </section>

  <section class="panel">
    <LevelSelector {selectedLevels} on:change={updateLevels} />
  </section>

  {#if currentCard}
    <StudyCard card={currentCard} {revealed} />
    <div class="actions">
      <button class="primary-button" type="button" on:click={() => (revealed = !revealed)}>
        {revealed ? 'Hide answer' : 'Reveal answer'}
      </button>
      <button class="secondary-button" type="button" on:click={nextCard}>Next card</button>
    </div>
  {:else}
    <section class="panel">
      <p>Select at least one JLPT level to start reviewing cards.</p>
    </section>
  {/if}
</main>
