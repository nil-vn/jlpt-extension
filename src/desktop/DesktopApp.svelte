<script lang="ts">
  import { Database, Monitor, UploadCloud } from '@lucide/svelte';
  import { Badge } from '../lib/components/ui/badge';
  import { Button } from '../lib/components/ui/button';
  import { Card } from '../lib/components/ui/card';
</script>

<main class="app-shell desktop-shell">
  <section class="hero">
    <div class="hero__header">
      <div>
        <Badge class="hero__eyebrow" variant="outline"><Monitor size={14} /> Wails v3 Phase 1</Badge>
        <h1>JLPT Study Companion for Windows</h1>
        <p>Desktop migration shell using Go, Wails v3, Svelte 5, and SQLite.</p>
      </div>
    </div>
  </section>

  <Card class="settings-grid">
    <div class="section-heading">
      <div>
        <h2><Database size={20} /> SQLite storage foundation</h2>
        <p>Phase 1 adds the Wails app entrypoint and a Go service that imports extension JSON datasets into SQLite.</p>
      </div>
    </div>

    <div class="setting-row">
      <span><UploadCloud size={16} /> JSON import UI arrives in the next phase</span>
      <Button disabled variant="outline">Import JSON dataset</Button>
    </div>
  </Card>
</main>
