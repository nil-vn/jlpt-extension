<script lang="ts">
  import { cn } from '../../../utils';

  let className = '';
  export { className as class };
</script>

<section class={cn('ui-card', className)} {...$$restProps}>
  <slot />
</section>
