<script lang="ts">
  type Props = {
    selectedLevels: string[];
    onchange: (levels: string[]) => void;
  };

  const levels = ['n5', 'n4', 'n3', 'n2', 'n1'];
  let { selectedLevels, onchange }: Props = $props();

  const toggle = (level: string): void => {
    const next = selectedLevels.includes(level)
      ? selectedLevels.filter((item) => item !== level)
      : [...selectedLevels, level];
    onchange(next);
  };
</script>

<div class="level-selector" aria-label="Chọn cấp độ JLPT">
  {#each levels as level}
    <button
      type="button"
      class:active={selectedLevels.includes(level)}
      class="chip"
      aria-pressed={selectedLevels.includes(level)}
      onclick={() => toggle(level)}
    >
      {level.toUpperCase()}
    </button>
  {/each}
</div>
