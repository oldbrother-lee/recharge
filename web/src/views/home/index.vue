<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import { useAppStore } from '@/store/modules/app';
import { useMessage } from 'naive-ui';
import { request } from '@/service/request';
import HeaderBanner from './modules/header-banner.vue';
import CardData from './modules/card-data.vue';
import LineChart from './modules/line-chart.vue';
import PieChart from './modules/pie-chart.vue';
import ProjectNews from './modules/project-news.vue';
import CreativityBanner from './modules/creativity-banner.vue';
import { getOperatorStatistics } from '@/api/statistics';
import { useAuthStore } from '@/store/modules/auth';

const appStore = useAppStore();
const message = useMessage();
const gap = computed(() => (appStore.isMobile ? 0 : 16));

const authStore = useAuthStore();
console.log('当前用户 roles:', authStore.userInfo.roles);

// 订单统计数据
const statisticsData = ref({
  total: {
    total: 0,
    yesterday: 0,
    today: 0
  },
  status: {
    processing: 0,
    success: 0,
    failed: 0
  },
  yesterday_status: {
    yesterday_processing: 0,
    yesterday_success: 0,
    yesterday_failed: 0
  },
  profit: {
    costAmount: 0,
    profitAmount: 0
  }
});

// 计算成功率
const successRate = computed(() => {
  const { success, failed } = statisticsData.value.status;
  const total = success + failed;
  if (total === 0) return 0;
  return Math.round((success / total) * 100);
});

// 计算今日与昨日的对比百分比
const todayVsYesterday = computed(() => {
  const today = statisticsData.value.total.today;
  const yesterday = statisticsData.value.total.yesterday;
  if (yesterday === 0) {
    return {
      diff: today - yesterday,
      percent: today === 0 ? 0 : 100,
      up: today > yesterday
    };
  }
  const diff = today - yesterday;
  const percent = Math.round((diff / yesterday) * 100);
  return {
    diff,
    percent: Math.abs(percent),
    up: diff >= 0
  };
});

// 获取订单统计概览数据
const fetchOrderStatistics = async () => {
  try {
    const { data, error } = await request({
      url: '/statistics/order/realtime',
      method: 'GET'
    });
    if (data && !error) {
      statisticsData.value = data;
    }
  } catch (error) {
    message.error('获取订单统计数据失败');
    console.error('获取订单统计数据失败:', error);
  }
};

interface OperatorStat {
  operator: string;
  totalOrders: number;
}

const operatorStats = ref<OperatorStat[]>([]);

onMounted(async () => {
  const res = await getOperatorStatistics();
  operatorStats.value = (res.data || []).map((item: { isp: number; totalOrders: number }) => ({
    operator: item.isp === 1 ? '移动' : item.isp === 2 ? '电信' : '联通',
    totalOrders: item.totalOrders
  }));
  fetchOrderStatistics();
});
</script>

<template>
  <NSpace vertical :size="16">
    <NGrid :x-gap="gap" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="24 12:m 8:l 6:xl">
        <CardData :statistics-data="statisticsData" />
      </NGridItem>
      <NGridItem span="24 s:24 m:12 l:8 xl:6">
        <PieChart :data="operatorStats" />
      </NGridItem>
    </NGrid>
  </NSpace>
</template>

<style scoped>
.text-sm {
  font-size: 0.875rem;
}
.text-gray-500 {
  color: #6b7280;
}
</style>
