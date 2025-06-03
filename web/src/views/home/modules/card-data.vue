<script setup lang="ts">
import { computed } from 'vue';
import { createReusableTemplate } from '@vueuse/core';
import { $t } from '@/locales';

defineOptions({
  name: 'CardData'
});

// 定义 props 接收父组件传递的数据
const props = defineProps<{
  statisticsData: {
    total: {
      total: number;
      yesterday: number;
      today: number;
    };
    status: {
      processing: number;
      success: number;
      failed: number;
    };
    yesterday_status: {
      yesterday_processing: number;
      yesterday_success: number;
      yesterday_failed: number;
    };
    profit: {
      costAmount: number;
      profitAmount: number;
    };
  };
}>();

interface CardData {
  title: string;
  value: number | string;
  subValue?: string;
  icon: string;
  color: string;
  isUp?: boolean;
}

const cardData = computed<CardData[]>(() => [
  {
    title: $t('page.home.totalOrders'),
    value: props.statisticsData.total.total,
    icon: 'mdi:cart-outline',
    color: '#409eff'
  },
  {
    title: $t('page.home.todayOrders'),
    value: props.statisticsData.total.today,
    icon: 'mdi:cart-outline',
    color: '#67c23a'
  },
  {
    title: $t('page.home.yesterdayOrders'),
    value: props.statisticsData.total.yesterday,
    icon: 'mdi:calendar-clock',
    color: '#e6a23c'
  },
  {
    title: $t('page.home.costAmount'),
    value: props.statisticsData.profit.costAmount,
    icon: 'mdi:currency-cny',
    color: '#909399'
  },
  {
    title: $t('page.home.profit'),
    value: props.statisticsData.profit.profitAmount,
    icon: 'mdi:currency-cny',
    color: '#f56c6c'
  },
  {
    title: $t('page.home.processingOrders'),
    value: props.statisticsData.status.processing,
    icon: 'mdi:clock-outline',
    color: '#409eff'
  },
  {
    title: $t('page.home.successOrders'),
    value: props.statisticsData.status.success,
    icon: 'mdi:check-circle-outline',
    color: '#67c23a'
  },
  {
    title: $t('page.home.failedOrders'),
    value: props.statisticsData.status.failed,
    icon: 'mdi:close-circle-outline',
    color: '#f56c6c'
  }
]);

// 计算今日与昨日的对比百分比
const todayVsYesterday = computed(() => {
  const today = props.statisticsData.total.today;
  const yesterday = props.statisticsData.total.yesterday;
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

// 计算今日与昨日成功订单的对比百分比
const successTodayVsYesterday = computed(() => {
  const today = props.statisticsData.status.success;
  const yesterday = props.statisticsData.yesterday_status.yesterday_success; // 这里需要后端提供昨日成功订单数
  console.log('successTodayVsYesterday', today, yesterday);
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

// 计算今日与昨日失败订单的对比百分比
const failedTodayVsYesterday = computed(() => {
  const today = props.statisticsData.status.failed;
  const yesterday = props.statisticsData.yesterday_status.yesterday_failed; // 这里需要后端提供昨日失败订单数
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

interface GradientBgProps {
  gradientColor: string;
}

const [DefineGradientBg, GradientBg] = createReusableTemplate<GradientBgProps>();

function getGradientColor(color: string) {
  return `linear-gradient(to bottom right, ${color}, ${color})`;
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper" title="订单统计概览">
    <!-- define component start: GradientBg -->
    <DefineGradientBg v-slot="{ $slots, gradientColor }">
      <div class="rd-8px px-16px pb-4px pt-8px text-white" :style="{ backgroundImage: gradientColor }">
        <component :is="$slots.default" />
      </div>
    </DefineGradientBg>
    <!-- define component end: GradientBg -->

    <NGrid cols="s:1 m:2 l:5" responsive="screen" :x-gap="16" :y-gap="16">
      <NGi v-for="item in cardData" :key="item.title">
        <GradientBg :gradient-color="getGradientColor(item.color)" class="flex-1">
          <h3 class="text-16px">{{ item.title }}</h3>
          <div class="flex justify-between pt-12px">
            <SvgIcon :icon="item.icon" class="text-32px" />
            <div class="text-30px text-white dark:text-dark">{{ item.value }}</div>
          </div>
        </GradientBg>
      </NGi>
    </NGrid>

    <!-- 新增对比卡片 -->
    <NGrid cols="3" responsive="screen" :x-gap="16" :y-gap="16" class="mt-4">
      <!-- 今日/昨日订单对比 -->
      <NGi>
        <GradientBg :gradient-color="getGradientColor('#67c23a')" class="flex-1">
          <h3 class="text-16px">今日/昨日订单对比</h3>
          <div class="flex justify-between pt-12px">
            <SvgIcon icon="mdi:chart-line" class="text-32px" />
            <div class="flex flex-col items-end">
              <div class="text-30px text-white dark:text-dark">
                {{ props.statisticsData.total.today }}/{{ props.statisticsData.total.yesterday }}
              </div>
              <div class="text-16px mt-1" :class="{ 'text-green-400': todayVsYesterday.up, 'text-red-400': !todayVsYesterday.up }">
                {{ todayVsYesterday.up ? '↑' : '↓' }} {{ todayVsYesterday.percent }}%
              </div>
            </div>
          </div>
        </GradientBg>
      </NGi>

      <!-- 今日/昨日成功订单对比 -->
      <NGi>
        <GradientBg :gradient-color="getGradientColor('#67c23a')" class="flex-1">
          <h3 class="text-16px">今日/昨日成功订单对比</h3>
          <div class="flex justify-between pt-12px">
            <SvgIcon icon="mdi:check-circle-outline" class="text-32px" />
            <div class="flex flex-col items-end">
              <div class="text-30px text-white dark:text-dark">
                {{ props.statisticsData.status.success }}/{{ props.statisticsData.yesterday_status.yesterday_success }}
              </div>
              <div class="text-16px mt-1" :class="{ 'text-green-400': successTodayVsYesterday.up, 'text-red-400': !successTodayVsYesterday.up }">
                {{ successTodayVsYesterday.up ? '↑' : '↓' }} {{ successTodayVsYesterday.percent }}%
              </div>
            </div>
          </div>
        </GradientBg>
      </NGi>

      <!-- 今日/昨日失败订单对比 -->
      <NGi>
        <GradientBg :gradient-color="getGradientColor('#f56c6c')" class="flex-1">
          <h3 class="text-16px">今日/昨日失败订单对比</h3>
          <div class="flex justify-between pt-12px">
            <SvgIcon icon="mdi:close-circle-outline" class="text-32px" />
            <div class="flex flex-col items-end">
              <div class="text-30px text-white dark:text-dark">
                {{ props.statisticsData.status.failed }}/{{ props.statisticsData.yesterday_status.yesterday_failed }}
              </div>
              <div class="text-16px mt-1" :class="{ 'text-green-400': failedTodayVsYesterday.up, 'text-red-400': !failedTodayVsYesterday.up }">
                {{ failedTodayVsYesterday.up ? '↑' : '↓' }} {{ failedTodayVsYesterday.percent }}%
              </div>
            </div>
          </div>
        </GradientBg>
      </NGi>
    </NGrid>
  </NCard>
</template>

<style scoped>
.text-green-400 {
  color: #4ade80;
}
.text-red-400 {
  color: #f87171;
}
</style>
