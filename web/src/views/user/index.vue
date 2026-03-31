<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import {
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NEmpty,
  NGrid,
  NGridItem,
  NSpace,
  NTag,
  NText,
  useMessage
} from 'naive-ui';
import { request } from '@/service/request';
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
  navigator.clipboard
    .writeText(text)
    .then(() => {
      message.success('已复制到剪贴板');
    })
    .catch(() => {
      message.error('复制失败');
    });
}

onMounted(() => {
  fetchProfile();
  fetchApiKeys();
});
</script>

<template>
  <NGrid :cols="24" :x-gap="16" :y-gap="16">
    <!-- 个人信息卡片 -->
    <NGridItem :span="24" :md="12">
      <NCard title="个人中心">
        <NDescriptions bordered :column="1">
          <NDescriptionsItem label="用户名">{{ userInfo.userName }}</NDescriptionsItem>
          <NDescriptionsItem label="余额">¥{{ userInfo.balance?.toFixed(2) || '0.00' }}</NDescriptionsItem>
          <NDescriptionsItem label="授信额度">¥{{ userInfo.credit?.toFixed(2) || '0.00' }}</NDescriptionsItem>
          <NDescriptionsItem label="状态">
            <NTag :type="userInfo.status === 1 ? 'success' : 'error'">
              {{ userInfo.status === 1 ? '正常' : '禁用' }}
            </NTag>
          </NDescriptionsItem>
          <NDescriptionsItem label="创建时间">
            {{ userInfo.created_at ? new Date(userInfo.created_at).toLocaleString() : '' }}
          </NDescriptionsItem>
        </NDescriptions>
      </NCard>
    </NGridItem>

    <!-- API密钥管理卡片 -->
    <NGridItem :span="24" :md="12">
      <NCard title="API密钥管理">
        <div v-if="!apiKey">
          <NEmpty description="暂无API密钥">
            <template #extra>
              <NButton type="primary" :loading="generating" @click="generateApiKey">生成API密钥</NButton>
            </template>
          </NEmpty>
        </div>
        <div v-else>
          <NCard size="small">
            <NDescriptions bordered :column="1" size="small">
              <NDescriptionsItem label="App ID">
                <NSpace align="center">
                  <NText code>{{ apiKey.app_id }}</NText>
                  <NButton text @click="copyToClipboard(apiKey.app_id)">
                    <template #icon>
                      <icon-ic-round-content-copy />
                    </template>
                  </NButton>
                </NSpace>
              </NDescriptionsItem>
              <NDescriptionsItem label="App Key">
                <NSpace align="center">
                  <NText code>{{ apiKey.app_key }}</NText>
                  <NButton text @click="copyToClipboard(apiKey.app_key)">
                    <template #icon>
                      <icon-ic-round-content-copy />
                    </template>
                  </NButton>
                </NSpace>
              </NDescriptionsItem>
              <NDescriptionsItem label="App Secret">
                <NSpace align="center">
                  <NText code>{{ showSecret ? apiKey.app_secret : '••••••••••••••••' }}</NText>
                  <NButton text @click="toggleSecret()">
                    <template #icon>
                      <icon-ic-round-visibility v-if="!showSecret" />
                      <icon-ic-round-visibility-off v-else />
                    </template>
                  </NButton>
                  <NButton text @click="copyToClipboard(apiKey.app_secret)">
                    <template #icon>
                      <icon-ic-round-content-copy />
                    </template>
                  </NButton>
                </NSpace>
              </NDescriptionsItem>
              <NDescriptionsItem label="状态">
                <NTag :type="apiKey.status === 1 ? 'success' : 'error'">
                  {{ apiKey.status === 1 ? '启用' : '禁用' }}
                </NTag>
              </NDescriptionsItem>
              <NDescriptionsItem label="创建时间">
                {{ apiKey.created_at ? new Date(apiKey.created_at).toLocaleString() : '' }}
              </NDescriptionsItem>
            </NDescriptions>
            <template #action>
              <NSpace>
                <NButton size="small" :loading="regenerating" @click="regenerateApiKey()">重新生成</NButton>
                <NButton
                  size="small"
                  :type="apiKey.status === 1 ? 'error' : 'success'"
                  :loading="toggling"
                  @click="toggleStatus(apiKey.status)"
                >
                  {{ apiKey.status === 1 ? '禁用' : '启用' }}
                </NButton>
              </NSpace>
            </template>
          </NCard>
          <!--
 <n-space>
            <n-button @click="generateApiKey" :loading="generating">
              生成新的API密钥
            </n-button>
          </n-space> 
-->
        </div>
      </NCard>
    </NGridItem>
  </NGrid>
</template>

<style scoped>
/* 使用 NaiveUI 栅格布局，无需自定义样式 */
</style>
