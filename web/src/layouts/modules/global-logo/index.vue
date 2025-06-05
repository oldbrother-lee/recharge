<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { publicSystemApi } from '@/api/system';
import { $t } from '@/locales';

defineOptions({
  name: 'GlobalLogo'
});

interface Props {
  /** Whether to show the title */
  showTitle?: boolean;
}

withDefaults(defineProps<Props>(), {
  showTitle: true
});

const systemName = ref($t('system.title')); // 默认值
const systemLogo = ref(''); // 系统Logo

// 获取系统基本信息
const getSystemInfo = async () => {
  try {
    const response = await publicSystemApi.getSystemBasicInfo();
    if (response.data && response.data.configs) {
      if (response.data.configs.system_name) {
        systemName.value = response.data.configs.system_name;
      }
      if (response.data.configs.system_logo) {
        systemLogo.value = response.data.configs.system_logo;
      }
    }
  } catch (error) {
    console.warn('获取系统信息失败，使用默认值:', error);
  }
};

// 监听系统Logo更新事件
const handleLogoUpdate = (event: CustomEvent) => {
  console.log('收到Logo更新事件:', event.detail);
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
  <RouterLink to="/" class="w-full flex-center nowrap-hidden">
    <div class="logo-container">
      <img 
        v-if="systemLogo" 
        :src="systemLogo" 
        :alt="systemName"
        class="system-logo-image"
      />
      <SystemLogo v-else class="text-32px text-primary" />
    </div>
    <h2 v-show="showTitle" class="pl-8px text-16px text-primary font-bold transition duration-300 ease-in-out">
      {{ systemName }}
    </h2>
  </RouterLink>
</template>

<style scoped>
.logo-container {
  display: flex;
  align-items: center;
  justify-content: center;
}

.system-logo-image {
  width: 32px;
  height: 32px;
  object-fit: contain;
  border-radius: 4px;
}
</style>
