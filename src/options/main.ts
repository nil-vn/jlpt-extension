import { mount } from 'svelte';
import Options from './Options.svelte';

document.body.classList.add('options-body');

export default mount(Options, { target: document.getElementById('app')! });
