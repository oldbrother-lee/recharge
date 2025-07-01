<script setup lang="ts">
import { ref } from 'vue';
import { NForm, NFormItemGi, NInput, NSelect, NButton, NSpace, NDatePicker, NGrid, NCard, NCollapse, NCollapseItem } from 'naive-ui';
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
  <n-card :bordered="false" size="small" class="search-form-card">
    <n-form :model="searchForm" label-placement="left" :label-width="80" class="search-form">
      <n-collapse :default-expanded-names="[]">
        <n-collapse-item title="搜索条件" name="order-search">
          <n-grid responsive="screen" item-responsive :x-gap="24" class="search-grid">
            <n-form-item-gi span="24 s:12 m:6" label="订单号" path="order_number" class="pr-24px form-item">
              <n-input v-model:value="searchForm.order_number" placeholder="请输入订单号" class="form-input" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="外部订单" path="out_trade_num" class="pr-24px form-item">
              <n-input v-model:value="searchForm.out_trade_num" placeholder="请输入外部订单号" class="form-input" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="手机号" path="mobile" class="pr-24px form-item">
              <n-input v-model:value="searchForm.mobile" placeholder="请输入手机号" class="form-input" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="运营商" path="isp" class="pr-24px form-item">
              <n-select
                v-model:value="searchForm.isp"
                :options="[
                  { label: '移动', value: 1 },
                  { label: '联通', value: 3 },
                  { label: '电信', value: 2 },
                ]"
                placeholder="请选择运营商"
                clearable
                class="form-select"
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="订单状态" path="status" class="pr-24px form-item">
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
                class="form-select"
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="创建时间" path="date_range" class="pr-24px form-item">
              <n-date-picker v-model:value="searchForm.date_range" type="daterange" clearable class="form-date-picker" />
            </n-form-item-gi>
            <n-form-item-gi span="24" class="pr-24px form-item">
              <n-space class="w-full search-buttons" justify="end">
                <n-button @click="handleReset" class="search-btn">重置</n-button>
                <n-button type="primary" ghost @click="handleSearch" class="search-btn">搜索</n-button>
              </n-space>
            </n-form-item-gi>
          </n-grid>
        </n-collapse-item>
      </n-collapse>
    </n-form>
  </n-card>
</template>

<style scoped>
/* 基础样式 */
.search-form-card {
  margin: 0;
  padding: 0;
}

.search-form {
  width: 100%;
}

.search-grid {
  width: 100%;
}

.pr-24px {
  padding-right: 24px;
}

.w-full {
  width: 100%;
}

.search-buttons {
  flex-wrap: wrap;
}

.search-btn {
  min-width: 80px;
}

.form-item {
  margin-bottom: 16px;
}

.form-input,
.form-select,
.form-date-picker {
  width: 100%;
  min-width: 0;
}

/* 平板端优化 (640px-768px) */
@media (max-width: 768px) {
  .search-form-card {
    margin: 0 8px;
  }
  
  .pr-24px {
    padding-right: 16px;
  }
  
  .search-grid {
    gap: 16px;
  }
}

/* 移动端优化 (480px-640px) */
@media (max-width: 640px) {
  .search-form-card {
    margin: 0 4px;
    padding: 12px;
  }
  
  .search-form {
    font-size: 14px;
  }
  
  .pr-24px {
    padding-right: 8px;
  }
  
  .search-grid {
    gap: 12px;
  }
  
  .form-item {
    margin-bottom: 12px;
  }
  
  .search-buttons {
    justify-content: center !important;
    gap: 12px;
    margin-top: 8px;
  }
  
  .search-btn {
    flex: 1;
    max-width: 120px;
    min-width: 100px;
    font-size: 14px;
  }
  
  .form-input,
  .form-select,
  .form-date-picker {
    font-size: 14px;
  }
}

/* 极小屏幕优化 (320px-480px) */
@media (max-width: 480px) {
  .search-form-card {
    margin: 0;
    padding: 8px;
  }
  
  .search-form {
    font-size: 13px;
  }
  
  .pr-24px {
    padding-right: 4px;
  }
  
  .search-grid {
    gap: 8px;
  }
  
  .form-item {
    margin-bottom: 8px;
  }
  
  .search-buttons {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    margin-top: 12px;
  }
  
  .search-btn {
    width: 100%;
    max-width: none;
    font-size: 13px;
    padding: 8px 16px;
  }
  
  .form-input,
  .form-select,
  .form-date-picker {
    font-size: 13px;
    min-height: 32px;
  }
  
  /* 确保表单项在极小屏幕上占满宽度 */
  :deep(.n-form-item-gi) {
    width: 100% !important;
    flex: none !important;
  }
  
  /* 优化标签显示 */
  :deep(.n-form-item-label) {
    font-size: 13px;
    min-width: 60px !important;
    width: 60px !important;
  }
  
  /* 优化输入框内边距 */
  :deep(.n-input__input-el),
  :deep(.n-base-selection-input) {
    padding: 6px 8px !important;
  }
}

/* 超小屏幕优化 (小于320px) */
@media (max-width: 320px) {
  .search-form-card {
    padding: 4px;
  }
  
  .pr-24px {
    padding-right: 2px;
  }
  
  :deep(.n-form-item-label) {
    min-width: 50px !important;
    width: 50px !important;
    font-size: 12px;
  }
  
  .form-input,
  .form-select,
  .form-date-picker {
    font-size: 12px;
  }
  
  .search-btn {
    font-size: 12px;
    padding: 6px 12px;
  }
}
</style>