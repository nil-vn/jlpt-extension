<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { jlptLevels } from '../data';
  import type { JlptLevel } from '../types/flashcard';

  export let selectedLevels: JlptLevel[] = ['N5'];

  const dispatch = createEventDispatcher<{ change: JlptLevel[] }>();

  function toggleLevel(level: JlptLevel) {
    selectedLevels = selectedLevels.includes(level)
      ? selectedLevels.filter((selectedLevel) => selectedLevel !== level)
      : [...selectedLevels, level];

    dispatch('change', selectedLevels);
  }
</script>

<div class="level-selector" aria-label="JLPT levels">
  {#each jlptLevels as level}
    <button
      type="button"
      class:active={selectedLevels.includes(level)}
      aria-pressed={selectedLevels.includes(level)}
      on:click={() => toggleLevel(level)}
    >
      {level}
    </button>
  {/each}
</div>
