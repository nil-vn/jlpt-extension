<script lang="ts">
  import '../app.css';
  import { LevelSelector } from '../lib/components';
  import { levelDescriptions } from '../lib/data';
  import type { JlptLevel, StudySettings } from '../lib/types/flashcard';

  let settings: StudySettings = {
    dailyGoal: 20,
    selectedLevels: ['N5', 'N4'],
    enabledCategories: ['vocabulary', 'kanji', 'grammar']
  };

  function updateLevels(event: CustomEvent<JlptLevel[]>) {
    settings = { ...settings, selectedLevels: event.detail };
  }
</script>

<main class="app-shell">
  <section class="hero">
    <h1>Study settings</h1>
    <p>Tune your extension defaults for daily JLPT practice.</p>
  </section>

  <section class="panel settings-grid">
    <label class="setting-row">
      <span>Daily review goal</span>
      <input bind:value={settings.dailyGoal} min="1" type="number" />
    </label>

    <div class="setting-row">
      <span>Active JLPT levels</span>
      <LevelSelector selectedLevels={settings.selectedLevels} on:change={updateLevels} />
    </div>
  </section>

  <section class="panel settings-grid">
    <h2>Level guide</h2>
    {#each settings.selectedLevels as level}
      <article>
        <strong>{level}</strong>
        <p>{levelDescriptions[level]}</p>
      </article>
    {/each}
  </section>
</main>
