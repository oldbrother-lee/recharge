<template>
  <n-card title="个人中心" class="profile-card">
    <n-descriptions bordered :column="1">
      <n-descriptions-item label="用户名">{{ userInfo.userName }}</n-descriptions-item>
      <!-- <n-descriptions-item label="昵称">{{ userInfo.nickname }}</n-descriptions-item>
      <n-descriptions-item label="邮箱">{{ userInfo.email }}</n-descriptions-item>
      <n-descriptions-item label="手机号">{{ userInfo.phone }}</n-descriptions-item> -->
      <n-descriptions-item label="余额">¥{{ userInfo.balance?.toFixed(2) || '0.00' }}</n-descriptions-item>
      <n-descriptions-item label="授信额度">¥{{ userInfo.credit?.toFixed(2) || '0.00' }}</n-descriptions-item>
      <n-descriptions-item label="状态">{{ userInfo.status === 1 ? '正常' : '禁用' }}</n-descriptions-item>
      <n-descriptions-item label="创建时间">{{ userInfo.created_at ? new Date(userInfo.created_at).toLocaleString() : '' }}</n-descriptions-item>
    </n-descriptions>
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { request } from '@/service/request';
import { useMessage, NCard, NDescriptions, NDescriptionsItem } from 'naive-ui';

const message = useMessage();
const userInfo = ref<any>({});

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

onMounted(() => {
  fetchProfile();
});
</script>

<style scoped>
.profile-card {
  max-width: 480px;
  margin: 32px auto;
}
</style> 