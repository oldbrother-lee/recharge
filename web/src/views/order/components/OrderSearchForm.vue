<script setup lang="ts">
import { computed, toRaw } from 'vue';
import { NForm, NFormItemGi, NInput, NSelect, NButton, NSpace, NDatePicker, NGrid, NCard, NCollapse, NCollapseItem } from 'naive-ui';

defineOptions({
  name: 'OrderSearch'
});

interface Emits {
  (e: 'search'): void;
}

const emit = defineEmits<Emits>();

const model = defineModel<any>('model', { required: true });

const defaultModel = computed(() => ({ ...toRaw(model.value) }));

function resetModel() {
  Object.assign(model.value, defaultModel.value);
}

async function reset() {
  resetModel();
  emit('search');
}

async function search() {
  emit('search');
}
</script>
<template>
  <n-card :bordered="false" size="small" class="card-wrapper">
    <n-form :model="model" label-placement="left" :label-width="80">
      <n-collapse>
        <n-collapse-item title="搜索条件" name="order-search">
          <n-grid responsive="screen" item-responsive>
            <n-form-item-gi span="24 s:12 m:6" label="订单号" path="order_number" class="pr-24px form-item">
              <n-input v-model:value="model.order_number" placeholder="请输入订单号" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="外部订单" path="out_trade_num" class="pr-24px form-item">
              <n-input v-model:value="model.out_trade_num" placeholder="请输入外部订单号" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="手机号" path="mobile" class="pr-24px form-item">
              <n-input v-model:value="model.mobile" placeholder="请输入手机号" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="运营商" path="isp" class="pr-24px form-item">
              <n-select
                v-model:value="model.isp"
                :options="[
                  { label: '移动', value: 1 },
                  { label: '联通', value: 3 },
                  { label: '电信', value: 2 },
                ]"
                placeholder="请选择运营商"
                clearable
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="面值" path="denom" class="pr-24px form-item">
              <n-select
                v-model:value="model.denom"
                :options="[
                  { label: '10', value: 10 },
                  { label: '20', value: 20 },
                  { label: '30', value: 30 },
                  { label: '50', value: 50 },
                  { label: '100', value: 100 },
                  { label: '200', value: 200 },
                  { label: '300', value: 300 },
                  { label: '500', value: 500 }
                ]"
                placeholder="请选择面值"
                clearable
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="订单状态" path="status" class="pr-24px form-item">
              <n-select
                v-model:value="model.status"
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
            <n-form-item-gi span="24 s:12 m:6" label="创建时间" path="date_range" class="pr-24px form-item">
              <n-date-picker v-model:value="model.date_range" type="daterange" clearable />
            </n-form-item-gi>
            <n-form-item-gi span="24" class="pr-24px form-item">
              <n-space class="w-full" justify="end">
                <n-button @click="reset">重置</n-button>
                <n-button type="primary" ghost @click="search">搜索</n-button>
              </n-space>
            </n-form-item-gi>
          </n-grid>
        </n-collapse-item>
      </n-collapse>
    </n-form>
  </n-card>
</template>
