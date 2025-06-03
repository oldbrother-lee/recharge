<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import { $t } from '@/locales';

interface DailyStatistics {
  date: string;
  totalOrders: number;
  successOrders: number;
  failedOrders: number;
  successRate: number;
  costAmount: number;
  profitAmount: number;
}

const chartData = ref<DailyStatistics[]>([]);

const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: {
    trigger: 'axis',
    axisPointer: {
      type: 'cross',
      label: {
        backgroundColor: '#6a7985'
      }
    }
  },
  legend: {
    data: [
      $t('page.home.totalOrders'),
      $t('page.home.successOrders'),
      $t('page.home.failedOrders'),
      $t('page.home.costAmount'),
      $t('page.home.profit')
    ]
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: [] as string[]
  },
  yAxis: [
    {
      type: 'value',
      name: $t('page.home.totalOrders'),
      position: 'left',
      min: 0
    },
    {
      type: 'value',
      name: $t('page.home.costAmount'),
      position: 'right',
      min: 0
    }
  ],
  series: [
    {
      name: $t('page.home.totalOrders'),
      type: 'line',
      data: [] as number[],
      lineStyle: { color: '#409eff' },
      showSymbol: true
    },
    {
      name: $t('page.home.successOrders'),
      type: 'line',
      data: [] as number[],
      lineStyle: { color: '#67c23a' },
      showSymbol: true
    },
    {
      name: $t('page.home.failedOrders'),
      type: 'line',
      data: [] as number[],
      lineStyle: { color: '#f56c6c' },
      showSymbol: true
    },
    {
      name: $t('page.home.costAmount'),
      type: 'line',
      yAxisIndex: 1,
      data: [] as number[],
      lineStyle: { color: '#e6a23c' },
      showSymbol: true
    },
    {
      name: $t('page.home.profit'),
      type: 'line',
      yAxisIndex: 1,
      data: [] as number[],
      lineStyle: { color: '#909399' },
      showSymbol: true
    }
  ]
}));

const updateChart = (data: DailyStatistics[]) => {
  console.log('updateChart called', data);
  chartData.value = data;
  
  updateOptions(opts => {
    opts.xAxis.data = data.map(item => {
      const date = new Date(item.date);
      return `${date.getMonth() + 1}/${date.getDate()}`;
    });
    
    opts.series[0].data = data.map(item => item.totalOrders);
    opts.series[1].data = data.map(item => item.successOrders);
    opts.series[2].data = data.map(item => item.failedOrders);
    opts.series[3].data = data.map(item => item.costAmount);
    opts.series[4].data = data.map(item => item.profitAmount);

    console.log('opts for echarts', opts);
    return opts;
  });
};

// 模拟数据
const mockData: DailyStatistics[] = [
  { date: "2025-05-10T00:00:00+08:00", totalOrders: 10, successOrders: 5, failedOrders: 5, successRate: 0.5, costAmount: 100, profitAmount: 20 },
  { date: "2025-05-11T00:00:00+08:00", totalOrders: 20, successOrders: 10, failedOrders: 10, successRate: 0.5, costAmount: 200, profitAmount: 40 },
  {
    date: "2025-05-12T00:00:00+08:00",
    totalOrders: 36,
    successOrders: 0,
    failedOrders: 34,
    successRate: 0,
    costAmount: 1800,
    profitAmount: 0
  },
  {
    date: "2025-05-13T00:00:00+08:00",
    totalOrders: 15,
    successOrders: 0,
    failedOrders: 15,
    successRate: 0,
    costAmount: 750,
    profitAmount: 0
  },
  {
    date: "2025-05-14T00:00:00+08:00",
    totalOrders: 8,
    successOrders: 0,
    failedOrders: 8,
    successRate: 0,
    costAmount: 400,
    profitAmount: 0
  },
  {
    date: "2025-05-15T00:00:00+08:00",
    totalOrders: 12,
    successOrders: 0,
    failedOrders: 11,
    successRate: 0,
    costAmount: 600,
    profitAmount: 0
  },
  {
    date: "2025-05-16T00:00:00+08:00",
    totalOrders: 34,
    successOrders: 0,
    failedOrders: 32,
    successRate: 0,
    costAmount: 1700,
    profitAmount: 0
  },
  {
    date: "2025-05-17T00:00:00+08:00",
    totalOrders: 12,
    successOrders: 0,
    failedOrders: 12,
    successRate: 0,
    costAmount: 600,
    profitAmount: 0
  },
  {
    date: "2025-05-18T00:00:00+08:00",
    totalOrders: 12,
    successOrders: 0,
    failedOrders: 10,
    successRate: 0,
    costAmount: 520,
    profitAmount: 20
  },
  {
    date: "2025-05-19T00:00:00+08:00",
    totalOrders: 0,
    successOrders: 0,
    failedOrders: 0,
    successRate: 0,
    costAmount: 0,
    profitAmount: 0
  },
  {
    date: "2025-05-20T00:00:00+08:00",
    totalOrders: 12,
    successOrders: 0,
    failedOrders: 8,
    successRate: 0,
    costAmount: 600,
    profitAmount: 0
  }
];

onMounted(() => {
  setTimeout(() => {
    console.log('onMounted called');
    updateChart(mockData);
  }, 100);
});
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper" title="订单趋势">
    <div ref="domRef" class="h-400px"></div>
  </NCard>
</template>

<style scoped></style>
