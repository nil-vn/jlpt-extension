<script lang="ts">
  import type { Flashcard } from '../types/flashcard';

  type PlannedFlashcard = Partial<Flashcard> & {
    level?: string;
    category?: string;
  };

  const categoryLabels: Record<string, string> = {
    gramma: 'Ngữ pháp',
    locabulary: 'Từ vựng',
    kanji: 'Kanji',
    reading: 'Đọc hiểu',
    listening: 'Nghe hiểu'
  };

  export let card: PlannedFlashcard;
  export let revealed = false;

  $: categoryLabel = categoryLabels[String(card.category).toLowerCase()] ?? card.category;
  $: levelLabel = card.level ? card.level.toUpperCase() : '';
  $: cardName = card.name ?? '';
  $: cardMeaning = card.mean ?? '';
  $: cardReading = card.hiragana ?? '';

  function playAudio() {
    if (!card.audio) return;

    const audio = new Audio(card.audio);
    audio.play();
  }
</script>

<article class="study-card" aria-live="polite">
  <div class="study-card__meta">
    <span>{levelLabel}</span>
    <span>{categoryLabel}</span>
  </div>

  <h2>{cardName}</h2>

  {#if cardReading}
    <p class="study-card__reading">{cardReading}</p>
  {/if}

  {#if card.image}
    <img class="study-card__image" src={card.image} alt={`Minh họa cho ${cardName}`} />
  {/if}

  {#if card.audio}
    <button class="audio-button" type="button" on:click={playAudio} aria-label="Nghe phát âm">
      ▶ Nghe audio
    </button>
  {/if}

  {#if revealed}
    <p class="study-card__answer">{cardMeaning}</p>
    {#if card.example}
      <p class="study-card__example">{card.example}</p>
    {/if}
  {:else}
    <p class="study-card__hint">Bấm “Hiện đáp án” để kiểm tra nghĩa của thẻ.</p>
  {/if}
</article>
