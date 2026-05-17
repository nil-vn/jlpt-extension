import { mount } from 'svelte'
import Options from './Options.svelte'

document.body.classList.add('options-body')

const app = mount(Options, { target: document.getElementById('app')! })

export default app
