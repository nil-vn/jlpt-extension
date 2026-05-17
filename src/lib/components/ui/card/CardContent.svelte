<script lang="ts">
  import { cn } from '../../../utils';

  let className = '';
  export { className as class };
</script>

<div class={cn('ui-card__content', className)} {...$$restProps}>
  <slot />
</div>
