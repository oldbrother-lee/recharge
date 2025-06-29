<script setup lang="ts">
import { ref } from 'vue';
import { NForm, NFormItemGi, NInput, NSelect, NButton, NSpace, NDatePicker, NGrid } from 'naive-ui';
const emit = defineEmits(['search']);
const searchForm = ref({
  order_number: '',
  out_trade_num: '',
  mobile: '',
  status: null,
  platform_code: null,
  date_range: null
});
const handleSearch = () => {
  emit('search', { ...searchForm.value });
};
const handleReset = () => {
  searchForm.value = {
    order_number: '',
    out_trade_num: '',
    mobile: '',
    status: null,
    platform_code: null,
    date_range: null
  };
  emit('search', { ...searchForm.value });
};
</script>
<template>
  <n-form :model="searchForm" label-placement="left" :label-width="80">
    <n-grid responsive="screen" item-responsive>
      <n-form-item-gi span="24 s:12 m:6" label="订单号" path="order_number" class="pr-24px">
        <n-input v-model:value="searchForm.order_number" placeholder="请输入订单号" />
      </n-form-item-gi>
      <n-form-item-gi span="24 s:12 m:6" label="外部订单" path="out_trade_num" class="pr-24px">
        <n-input v-model:value="searchForm.out_trade_num" placeholder="请输入外部订单号" />
      </n-form-item-gi>
      <n-form-item-gi span="24 s:12 m:6" label="手机号" path="mobile" class="pr-24px">
        <n-input v-model:value="searchForm.mobile" placeholder="请输入手机号" />
      </n-form-item-gi>
      <n-form-item-gi span="24 s:12 m:6" label="运营商" path="isp" class="pr-24px">
        <n-select
          v-model:value="searchForm.isp"
          :options="[
            { label: '移动', value: 1 },
            { label: '联通', value: 3 },
            { label: '电信', value: 2 },
          ]"
          placeholder="请选择运营商"
          clearable
        />
      </n-form-item-gi>
      <n-form-item-gi span="24 s:12 m:6" label="订单状态" path="status" class="pr-24px">
        <n-select
          v-model:value="searchForm.status"
          :options="[
            { label: '待支付', value: 1 },
            { label: '待充值', value: 2 },
            { label: '充值中', value: 3 },
            { label: '充值成功', value: 4 },
            { label: '充值失败', value: 5 },
            { label: '已退款', value: 6 },
            { label: '已取消', value: 7 },
            { label: '部分充值', value: 8 },
            { label: '已拆单', value: 9 }
          ]"
          placeholder="请选择状态"
          clearable
          
        />
      </n-form-item-gi>

      <n-form-item-gi span="24 s:12 m:6" label="创建时间" path="date_range" class="pr-24px">
        <n-date-picker v-model:value="searchForm.date_range" type="daterange" clearable />
      </n-form-item-gi>
      <n-form-item-gi span="24 m:12" class="pr-24px">
        <n-space class="w-full" justify="end">
          <n-button @click="handleReset">重置</n-button>
          <n-button type="primary" ghost @click="handleSearch">搜索</n-button>
        </n-space>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template> 