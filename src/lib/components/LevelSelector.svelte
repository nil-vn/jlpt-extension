<script lang="ts">
  import { Check } from '@lucide/svelte';
  import { createEventDispatcher } from 'svelte';
  import { Button } from './ui/button';
  import { jlptLevels } from '../data';
  import type { JlptLevel } from '../types/flashcard';

  export let selectedLevels: JlptLevel[] = ['n5'];

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
    <Button
      variant={selectedLevels.includes(level) ? 'default' : 'outline'}
      size="sm"
      class={selectedLevels.includes(level) ? 'active' : ''}
      aria-pressed={selectedLevels.includes(level)}
      on:click={() => toggleLevel(level)}
    >
      {#if selectedLevels.includes(level)}
        <Check size={14} strokeWidth={2.5} />
      {/if}
      {level.toUpperCase()}
    </Button>
  {/each}
</div>
