<script lang="ts">
  import { BookOpenCheck, Database, Monitor, Settings } from '@lucide/svelte';
  import { Badge } from '../lib/components/ui/badge';
  import { Button } from '../lib/components/ui/button';
  import { Card } from '../lib/components/ui/card';
  import SettingsView from './views/SettingsView.svelte';
  import StudyView from './views/StudyView.svelte';

  type DesktopRoute = 'study' | 'settings';

  let currentRoute: DesktopRoute = 'study';

  const routes: Array<{ id: DesktopRoute; label: string; description: string }> = [
    { id: 'study', label: 'Study', description: 'Review JLPT flashcards using the reused extension study UI.' },
    { id: 'settings', label: 'Settings', description: 'Manage dataset, levels, language, and reminder preferences.' }
  ];

  $: currentRouteDescription = routes.find((route) => route.id === currentRoute)?.description ?? '';

  function navigate(route: DesktopRoute) {
    currentRoute = route;
  }
</script>

<main class="app-shell desktop-shell">
  <section class="hero desktop-hero">
    <div class="hero__header">
      <div>
        <Badge class="hero__eyebrow" variant="outline"><Monitor size={14} /> Wails v3 Desktop</Badge>
        <h1>JLPT Study Companion for Windows</h1>
        <p>{currentRouteDescription}</p>
      </div>
    </div>
  </section>

  <Card class="desktop-router-card">
    <nav class="segmented-toggle desktop-router" aria-label="Desktop sections">
      <button
        class:active={currentRoute === 'study'}
        class="toggle-icon-button"
        type="button"
        aria-pressed={currentRoute === 'study'}
        on:click={() => navigate('study')}
      >
        <BookOpenCheck size={14} />
        <span>Study</span>
      </button>
      <button
        class:active={currentRoute === 'settings'}
        class="toggle-icon-button"
        type="button"
        aria-pressed={currentRoute === 'settings'}
        on:click={() => navigate('settings')}
      >
        <Settings size={14} />
        <span>Settings</span>
      </button>
    </nav>

    <p class="help-text"><Database size={14} /> Phase 2 keeps the frontend source at the repo root and reuses extension components inside desktop routes.</p>
  </Card>

  {#if currentRoute === 'study'}
    <StudyView openSettings={() => navigate('settings')} />
  {:else}
    <SettingsView />
  {/if}
</main>
