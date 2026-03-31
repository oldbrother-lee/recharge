<script setup lang="ts">
import { computed, toRaw } from 'vue';
import {
  NButton,
  NCard,
  NCollapse,
  NCollapseItem,
  NDatePicker,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NSelect,
  NSpace
} from 'naive-ui';
import { ORDER_STATUS_OPTIONS } from '@/constants/business';

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
  <NCard :bordered="false" size="small" class="card-wrapper">
    <NForm :model="model" label-placement="left" :label-width="80">
      <NCollapse>
        <NCollapseItem title="搜索条件" name="order-search">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:6" label="订单号" path="order_number" class="form-item pr-24px">
              <NInput v-model:value="model.order_number" placeholder="请输入订单号" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="外部订单" path="out_trade_num" class="form-item pr-24px">
              <NInput v-model:value="model.out_trade_num" placeholder="请输入外部订单号" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="手机号" path="mobile" class="form-item pr-24px">
              <NInput v-model:value="model.mobile" placeholder="请输入手机号" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="运营商" path="isp" class="form-item pr-24px">
              <NSelect
                v-model:value="model.isp"
                :options="[
                  { label: '移动', value: 1 },
                  { label: '联通', value: 3 },
                  { label: '电信', value: 2 }
                ]"
                placeholder="请选择运营商"
                clearable
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="面值" path="denom" class="form-item pr-24px">
              <NSelect
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
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="订单状态" path="status" class="form-item pr-24px">
              <NSelect
                v-model:value="model.status"
                :options="ORDER_STATUS_OPTIONS"
                placeholder="请选择状态"
                clearable
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="创建时间" path="date_range" class="form-item pr-24px">
              <NDatePicker v-model:value="model.date_range" type="daterange" clearable />
            </NFormItemGi>
            <NFormItemGi span="24" class="form-item pr-24px">
              <NSpace class="w-full" justify="end">
                <NButton @click="reset">重置</NButton>
                <NButton type="primary" ghost @click="search">搜索</NButton>
              </NSpace>
            </NFormItemGi>
          </NGrid>
        </NCollapseItem>
      </NCollapse>
    </NForm>
  </NCard>
</template>
