<script lang="ts">
  import { cn } from '../../../utils';

  let className = '';
  export { className as class };
</script>

<p class={cn('ui-card__description', className)} {...$$restProps}>
  <slot />
</p>
