<script setup lang="ts">
import { watch } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';

interface OperatorStat {
  operator: string;
  totalOrders: number;
}

const props = defineProps<{ data: OperatorStat[] }>();
const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: { trigger: 'item' },
  legend: { top: '5%', left: 'center' },
  series: [
    {
      name: '订单数',
      type: 'pie',
      radius: ['40%', '70%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 10, borderColor: '#fff', borderWidth: 2 },
      label: {
        show: true,
        position: 'outside',
        formatter: '{b}: {c}'
      },
      emphasis: { label: { show: true, fontSize: 18, fontWeight: 'bold' } },
      labelLine: { show: true },
      data: []
    }
  ]
}));

watch(
  () => props.data,
  (val) => {
    updateOptions(opts => {
      opts.series[0].data = val.map(item => ({
        name: item.operator,
        value: item.totalOrders
      }));
      return opts;
    });
  },
  { immediate: true }
);
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper" title="运营商订单分布">
    <div ref="domRef" class="h-400px"></div>
  </NCard>
</template>

<style scoped></style>
