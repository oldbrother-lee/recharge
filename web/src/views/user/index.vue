<template>
  <n-grid :cols="24" :x-gap="16" :y-gap="16">
    <!-- 个人信息卡片 -->
    <n-grid-item :span="24" :md="12">
      <n-card title="个人中心">
        <n-descriptions bordered :column="1">
          <n-descriptions-item label="用户名">{{ userInfo.userName }}</n-descriptions-item>
          <n-descriptions-item label="余额">¥{{ userInfo.balance?.toFixed(2) || '0.00' }}</n-descriptions-item>
          <n-descriptions-item label="授信额度">¥{{ userInfo.credit?.toFixed(2) || '0.00' }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="userInfo.status === 1 ? 'success' : 'error'">
              {{ userInfo.status === 1 ? '正常' : '禁用' }}
            </n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="创建时间">
            {{ userInfo.created_at ? new Date(userInfo.created_at).toLocaleString() : '' }}
          </n-descriptions-item>
        </n-descriptions>
      </n-card>
    </n-grid-item>

    <!-- API密钥管理卡片 -->
    <n-grid-item :span="24" :md="12">
      <n-card title="API密钥管理">
        <div v-if="!apiKey">
          <n-empty description="暂无API密钥">
            <template #extra>
              <n-button type="primary" @click="generateApiKey" :loading="generating">
                生成API密钥
              </n-button>
            </template>
          </n-empty>
        </div>
        <div v-else>
          <n-card size="small">
            <n-descriptions bordered :column="1" size="small">
              <n-descriptions-item label="App ID">
                <n-space align="center">
                  <n-text code>{{ apiKey.app_id }}</n-text>
                  <n-button text @click="copyToClipboard(apiKey.app_id)">
                    <template #icon>
                      <icon-ic-round-content-copy />
                    </template>
                  </n-button>
                </n-space>
              </n-descriptions-item>
              <n-descriptions-item label="App Key">
                <n-space align="center">
                  <n-text code>{{ apiKey.app_key }}</n-text>
                  <n-button text @click="copyToClipboard(apiKey.app_key)">
                    <template #icon>
                      <icon-ic-round-content-copy />
                    </template>
                  </n-button>
                </n-space>
              </n-descriptions-item>
              <n-descriptions-item label="App Secret">
                <n-space align="center">
                  <n-text code>{{ showSecret ? apiKey.app_secret : '••••••••••••••••' }}</n-text>
                  <n-button text @click="toggleSecret()">
                    <template #icon>
                      <icon-ic-round-visibility v-if="!showSecret" />
                      <icon-ic-round-visibility-off v-else />
                    </template>
                  </n-button>
                  <n-button text @click="copyToClipboard(apiKey.app_secret)">
                    <template #icon>
                      <icon-ic-round-content-copy />
                    </template>
                  </n-button>
                </n-space>
              </n-descriptions-item>
              <n-descriptions-item label="状态">
                <n-tag :type="apiKey.status === 1 ? 'success' : 'error'">
                  {{ apiKey.status === 1 ? '启用' : '禁用' }}
                </n-tag>
              </n-descriptions-item>
              <n-descriptions-item label="创建时间">
                {{ apiKey.created_at ? new Date(apiKey.created_at).toLocaleString() : '' }}
              </n-descriptions-item>
            </n-descriptions>
            <template #action>
              <n-space>
                <n-button size="small" @click="regenerateApiKey()" :loading="regenerating">
                  重新生成
                </n-button>
                <n-button 
                  size="small" 
                  :type="apiKey.status === 1 ? 'error' : 'success'"
                  @click="toggleStatus(apiKey.status)"
                  :loading="toggling"
                >
                  {{ apiKey.status === 1 ? '禁用' : '启用' }}
                </n-button>
              </n-space>
            </template>
          </n-card>
          <!-- <n-space>
            <n-button @click="generateApiKey" :loading="generating">
              生成新的API密钥
            </n-button>
          </n-space> -->
        </div>
      </n-card>
    </n-grid-item>
  </n-grid>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue';
import { request } from '@/service/request';
import { 
  useMessage, 
  NCard, 
  NDescriptions, 
  NDescriptionsItem, 
  NEmpty, 
  NButton, 
  NSpace, 
  NText, 
  NTag,
  NGrid,
  NGridItem
} from 'naive-ui';
import { createAPIKey, getMyAPIKeys, regenerateAPIKey, updateAPIKeyStatus } from '@/api/external-api-key';

const message = useMessage();
const userInfo = ref<any>({});
const apiKey = ref<any>(null);
const generating = ref(false);
const regenerating = ref(false);
const toggling = ref(false);
const showSecret = ref(false);

async function fetchProfile() {
  try {
    const res = await request({ url: '/user/profile', method: 'GET' });
    if (res.data) {
      userInfo.value = res.data;
    }
  } catch (error) {
    message.error('获取用户信息失败');
  }
}

async function fetchApiKeys() {
  try {
    const res = await getMyAPIKeys();
    apiKey.value = res.data;
  } catch (error) {
    console.error('获取API密钥失败:', error);
  }
}

async function generateApiKey() {
  generating.value = true;
  try {
    const res = await createAPIKey({ app_name: '默认密钥' });
    if (res.data) {
      message.success('API密钥生成成功');
      await fetchApiKeys();
    }
  } catch (error: any) {
    message.error(error.message || '生成API密钥失败');
  } finally {
    generating.value = false;
  }
}

async function regenerateApiKey() {
  if (!apiKey.value) return;
  regenerating.value = true;
  try {
    const res = await regenerateAPIKey(apiKey.value.id);
    if (res.data) {
      message.success('API密钥重新生成成功');
      await fetchApiKeys();
    }
  } catch (error: any) {
    message.error(error.message || '重新生成API密钥失败');
  } finally {
    regenerating.value = false;
  }
}

async function toggleStatus(currentStatus: number) {
  if (!apiKey.value) return;
  toggling.value = true;
  try {
    const newStatus = currentStatus === 1 ? 0 : 1;
    const res = await updateAPIKeyStatus(apiKey.value.id, { status: newStatus });
    if (res.data) {
      message.success(`API密钥已${newStatus === 1 ? '启用' : '禁用'}`);
      await fetchApiKeys();
    }
  } catch (error: any) {
    message.error(error.message || '更新API密钥状态失败');
  } finally {
    toggling.value = false;
  }
}

function toggleSecret() {
  showSecret.value = !showSecret.value;
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    message.success('已复制到剪贴板');
  }).catch(() => {
    message.error('复制失败');
  });
}

onMounted(() => {
  fetchProfile();
  fetchApiKeys();
});
</script>

<style scoped>
/* 使用 NaiveUI 栅格布局，无需自定义样式 */
</style>