<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { publicSystemApi } from '@/api/system';

defineOptions({ name: 'SystemLogo' });

const systemLogo = ref(''); // 系统Logo

// 获取系统基本信息
const getSystemInfo = async () => {
  try {
    const response = await publicSystemApi.getSystemBasicInfo();
    if (response.data && response.data.configs) {
      if (response.data.configs.system_logo) {
        systemLogo.value = response.data.configs.system_logo;
      }
    }
  } catch (error) {
    console.warn('获取系统Logo失败，使用默认值:', error);
  }
};

// 监听系统Logo更新事件
const handleLogoUpdate = (event: CustomEvent) => {
  console.log('SystemLogo收到Logo更新事件:', event.detail);
  if (event.detail?.systemLogo) {
    systemLogo.value = event.detail.systemLogo;
  } else {
    // 如果没有直接传递Logo数据，重新获取系统信息
    getSystemInfo();
  }
};

onMounted(() => {
  getSystemInfo();
  // 监听全局Logo更新事件
  window.addEventListener('system-logo-updated', handleLogoUpdate as EventListener);
});

onUnmounted(() => {
  // 清理事件监听器
  window.removeEventListener('system-logo-updated', handleLogoUpdate as EventListener);
});
</script>

<template>
  <img 
    v-if="systemLogo" 
    :src="systemLogo" 
    alt="系统Logo"
    class="system-logo-image"
  />
  <icon-local-logo v-else />
</template>

<style scoped>
.system-logo-image {
  width: 1em;
  height: 1em;
  object-fit: contain;
  border-radius: 4px;
}
</style>
