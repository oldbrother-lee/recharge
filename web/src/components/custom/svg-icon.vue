<script setup lang="ts">
import { computed, useAttrs } from 'vue';

defineOptions({ name: 'SvgIcon', inheritAttrs: false });

/**
 * Props
 *
 * - Support iconify and local svg icon
 * - If icon and localIcon are passed at the same time, localIcon will be rendered first
 */
interface Props {
  /** Iconify icon name */
  icon?: string;
  /** Local svg icon name */
  localIcon?: string;
}

const props = defineProps<Props>();

const attrs = useAttrs();

const bindAttrs = computed<{ class: string; style: string }>(() => ({
  class: (attrs.class as string) || '',
  style: (attrs.style as string) || ''
}));

const symbolId = computed(() => {
  const { VITE_ICON_LOCAL_PREFIX: prefix } = import.meta.env;

  const defaultLocalIcon = 'no-icon';

  const icon = props.localIcon || defaultLocalIcon;

  return `#${prefix}-${icon}`;
});

/** If localIcon is passed, render localIcon first */
const renderLocalIcon = computed(() => props.localIcon || !props.icon);

/**
 * Get UnoCSS icon class
 *
 * @example
 *   mdi:home -> icon-mdi-home (depends on VITE_ICON_PREFIX)
 */
const unoIconClass = computed(() => {
  if (!props.icon) return '';
  const { VITE_ICON_PREFIX: prefix } = import.meta.env;
  return `${prefix}-${props.icon.replace(/:/g, '-')}`;
});
</script>

<template>
  <template v-if="renderLocalIcon">
    <svg aria-hidden="true" width="1em" height="1em" v-bind="bindAttrs">
      <use :xlink:href="symbolId" fill="currentColor" />
    </svg>
  </template>
  <template v-else>
    <!-- 优先使用 UnoCSS Class 渲染，实现完全本地化 -->
    <span v-if="icon" :class="unoIconClass" v-bind="bindAttrs"></span>
  </template>
</template>

<style scoped></style>
