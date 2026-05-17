import Popup from './Popup.svelte';

const app = new Popup({
  target: document.getElementById('app') as HTMLElement
});

export default app;
