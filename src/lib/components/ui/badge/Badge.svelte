<script lang="ts">
  import { cn } from '../../../utils';

  export let variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning' = 'default';
  let className = '';
  export { className as class };
</script>

<span class={cn('ui-badge', `ui-badge--${variant}`, className)} {...$$restProps}>
  <slot />
</span>
