<script lang="ts">
  import { Headphones, Lightbulb, Volume2 } from '@lucide/svelte';
  import type { Flashcard } from '../types/flashcard';
  import { Badge } from './ui/badge';
  import { Button } from './ui/button';
  import { Card } from './ui/card';

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

<Card class="study-card" aria-live="polite">
  <div class="study-card__header">
    <div class="study-card__meta">
      <Badge>{levelLabel}</Badge>
      <Badge variant="secondary">{categoryLabel}</Badge>
    </div>
    <div class="study-card__header-actions">
      <slot name="header-actions" />
    </div>
  </div>

  <h2>{cardName}</h2>

  {#if cardReading}
    <p class="study-card__reading">{cardReading}</p>
  {/if}

  {#if card.image}
    <img class="study-card__image" src={card.image} alt={`Minh họa cho ${cardName}`} />
  {/if}

  {#if card.audio}
    <Button class="audio-button" variant="outline" size="sm" on:click={playAudio} aria-label="Nghe phát âm">
      <Volume2 size={16} />
      Nghe audio
    </Button>
  {/if}

  {#if revealed}
    <p class="study-card__answer"><Headphones size={18} /> {cardMeaning}</p>
    {#if card.example}
      <p class="study-card__example">{card.example}</p>
    {/if}
  {:else}
    <p class="study-card__hint"><Lightbulb size={16} /> Bấm “Hiện đáp án” để kiểm tra nghĩa của thẻ.</p>
  {/if}

  <slot name="footer" />
</Card>
