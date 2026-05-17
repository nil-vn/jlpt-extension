<script lang="ts">
  import { cn } from '../../../utils';

  export let variant: 'default' | 'secondary' | 'ghost' | 'outline' | 'destructive' = 'default';
  export let size: 'default' | 'sm' | 'icon' = 'default';
  export let type: 'button' | 'submit' | 'reset' = 'button';
  let className = '';
  export { className as class };
</script>

<button class={cn('ui-button', `ui-button--${variant}`, `ui-button--${size}`, className)} {type} {...$$restProps} on:click>
  <slot />
</button>
