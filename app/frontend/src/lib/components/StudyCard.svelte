<script lang="ts">
  import type { FlashcardDTO } from '../../../bindings/github.com/nil-vn/jlpt-extension/app/models';

  type Props = {
    card: FlashcardDTO | null;
    revealAnswers: boolean;
    onreveal: () => void;
    ontogglebookmark: () => void;
    onsavenote: (note: string) => void;
  };

  let { card, revealAnswers, onreveal, ontogglebookmark, onsavenote }: Props = $props();
  let draftNote = $state('');
  let lastCardID = $state<string | null>(null);

  $effect(() => {
    if (card?.id !== lastCardID) {
      lastCardID = card?.id ?? null;
      draftNote = card?.note ?? '';
    }
  });
</script>

<article class="study-card">
  {#if card}
    <div class="study-card__top">
      <div class="flashcard-top">
        <span>{card.level.toUpperCase()}</span>
        <span>{card.category === 'gramma' ? 'grammar' : card.category}</span>
      </div>
      <button type="button" class="icon-button" aria-pressed={card.bookmarked} onclick={ontogglebookmark}>
        {card.bookmarked ? '★ Bookmarked' : '☆ Bookmark'}
      </button>
    </div>

    <div class="study-card__prompt">
      <h3>{card.name}</h3>
      <p class="hiragana">{card.hiragana}</p>
    </div>

    {#if revealAnswers}
      <div class="answer-panel">
        <p>{card.mean}</p>
        {#if card.example}
          <small>{card.example}</small>
        {/if}
      </div>
    {:else}
      <button type="button" class="secondary reveal-button" onclick={onreveal}>Hiện đáp án</button>
    {/if}

    <label class="note-box">
      <span>Note cá nhân</span>
      <textarea bind:value={draftNote} placeholder="Ghi chú cách nhớ, ví dụ riêng..." rows="4"></textarea>
    </label>
    <div class="actions compact-actions">
      <button type="button" class="secondary" onclick={() => onsavenote(draftNote)}>Lưu note</button>
    </div>
  {:else}
    <p class="empty-state">Chưa có flashcard phù hợp. Import JSON hoặc nới filter level/category để bắt đầu học.</p>
  {/if}
</article>
