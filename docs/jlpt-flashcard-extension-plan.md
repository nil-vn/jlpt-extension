# JLPT Flashcard Extension Plan

## UI component approach

The initial UI prototype uses local Svelte primitives rather than adopting shadcn-svelte at this stage. Components such as `Button.svelte`, `Card.svelte`, `Badge.svelte`, and `Switch.svelte` should remain project-owned Svelte components that are inspired by shadcn-style composition, spacing, variants, and Tailwind-friendly class names.

This keeps the prototype lightweight while preserving the current JLPT visual direction. If the project later adopts shadcn-svelte, that migration should add the required setup files and dependencies in the same change that consumes them.

## `components.json`

Do not keep a `components.json` file unless the repository actively uses shadcn-svelte tooling or is intentionally preparing for a documented shadcn-svelte setup. Until then, local primitives are the source of truth for the prototype UI.
