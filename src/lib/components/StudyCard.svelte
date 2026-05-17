<script lang="ts">
  import type { Flashcard } from '../types/flashcard';

  export let card: Flashcard;
  export let revealed = false;
</script>

<article class="study-card" aria-live="polite">
  <div class="study-card__meta">
    <span>{card.level}</span>
    <span>{card.category}</span>
  </div>

  <h2>{card.prompt}</h2>

  {#if card.reading}
    <p class="study-card__reading">{card.reading}</p>
  {/if}

  {#if revealed}
    <p class="study-card__answer">{card.answer}</p>
    {#if card.example}
      <p class="study-card__example">{card.example}</p>
    {/if}
  {:else}
    <p class="study-card__hint">Reveal the card to check your answer.</p>
  {/if}
</article>
