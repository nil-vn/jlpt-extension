<script lang="ts">
  import { Headphones, Lightbulb, Volume2 } from '@lucide/svelte';
  import { translate, type AppLanguage } from '../i18n';
  import type { Flashcard } from '../types/flashcard';
  import { Badge } from './ui/badge';
  import { Button } from './ui/button';
  import { Card } from './ui/card';

  type PlannedFlashcard = Partial<Flashcard> & {
    level?: string;
    category?: string;
  };

  export let card: PlannedFlashcard;
  export let revealed = false;
  export let language: AppLanguage = 'en';

  $: levelLabel = card.level ? card.level.toUpperCase() : '';
  $: cardName = card.name ?? '';
  $: cardMeaning = card.mean ?? '';
  $: cardReading = card.hiragana ?? '';
  $: t = (key: Parameters<typeof translate>[1], params?: Parameters<typeof translate>[2]) => translate(language, key, params);

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
    <img class="study-card__image" src={card.image} alt={t('illustrationAlt', { name: cardName })} />
  {/if}

  {#if card.audio}
    <Button class="audio-button" variant="outline" size="sm" on:click={playAudio} aria-label={t('audioButtonLabel')}>
      <Volume2 size={16} />
      {t('audioButton')}
    </Button>
  {/if}

  {#if revealed}
    <p class="study-card__answer"><Headphones size={18} /> {cardMeaning}</p>
    {#if card.example}
      <p class="study-card__example">{card.example}</p>
    {/if}
  {:else}
    <p class="study-card__hint"><Lightbulb size={16} /> {t('studyCardHint')}</p>
  {/if}

  <slot name="footer" />
</Card>
