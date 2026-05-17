<script lang="ts">
  import { cn } from '../../../utils';

  let className = '';
  export { className as class };
</script>

<h2 class={cn('ui-card__title', className)} {...$$restProps}>
  <slot />
</h2>
