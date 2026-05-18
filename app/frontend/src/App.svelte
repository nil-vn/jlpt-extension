<script lang="ts">
  import { Events } from '@wailsio/runtime';
  import { AppService } from '../bindings/github.com/nil-vn/jlpt-extension/app';
  import type { AppStatus } from '../bindings/github.com/nil-vn/jlpt-extension/app/models';

  let message = $state('Xin chào từ Svelte 5');
  let response = $state('Đang chờ gọi Go service...');
  let status = $state<AppStatus | null>(null);
  let time = $state('Đang chờ event từ Wails...');

  const loadStatus = async (): Promise<void> => {
    status = await AppService.Status();
  };

  const callGoService = async (): Promise<void> => {
    response = await AppService.Echo(message);
  };

  loadStatus()
    .then(() => callGoService())
    .catch((error: unknown) => {
      response = error instanceof Error ? error.message : String(error);
    });

  Events.On('time', (timeValue: { data: string }) => {
    time = timeValue.data;
  });
</script>

<main class="shell">
  <section class="hero">
    <div>
      <p class="eyebrow">Milestone 0 spike</p>
      <h1>JLPT Flashcard Desktop</h1>
      <p class="lede">
        Skeleton desktop app dùng Golang, Wails v3 và Svelte 5. Màn hình này chứng minh frontend gọi được service Go qua Wails bindings.
      </p>
    </div>
    <div class="logo-mark" aria-hidden="true">日</div>
  </section>

  <section class="grid">
    <article class="card">
      <h2>Go service binding</h2>
      <label for="message">Message gửi sang Go</label>
      <div class="input-row">
        <input id="message" bind:value={message} autocomplete="off" />
        <button type="button" onclick={callGoService}>Call Go</button>
      </div>
      <p class="result">{response}</p>
    </article>

    <article class="card">
      <h2>Runtime status</h2>
      {#if status}
        <dl>
          <div><dt>App</dt><dd>{status.name}</dd></div>
          <div><dt>Runtime</dt><dd>{status.runtime}</dd></div>
          <div><dt>Frontend</dt><dd>{status.frontend}</dd></div>
          <div><dt>Storage</dt><dd>{status.storage}</dd></div>
          <div><dt>Import target</dt><dd>{status.importTarget}</dd></div>
          <div><dt>Started at</dt><dd>{status.startedAt}</dd></div>
        </dl>
      {:else}
        <p>Đang tải status từ Go...</p>
      {/if}
    </article>
  </section>

  <footer>{time}</footer>
</main>
