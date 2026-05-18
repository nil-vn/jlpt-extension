import DesktopApp from './DesktopApp.svelte';
import '../app.css';

const app = new DesktopApp({
  target: document.getElementById('app')!,
});

export default app;
