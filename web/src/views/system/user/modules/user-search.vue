<script setup lang="ts">
import { ref } from 'vue';
import {
  NButton,
  NCard,
  NCollapse,
  NCollapseItem,
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NInputNumber,
  NSelect,
  NSpace
} from 'naive-ui';

const model = defineModel<any>('model', { required: true });
const emit = defineEmits(['reset', 'search']);

function reset() {
  model.value = {
    user_name: '',
    phone: '',
    email: '',
    status: null,
    balance_min: null,
    balance_max: null
  };
  emit('reset');
}

function search() {
  emit('search');
}
</script>

<template>
  <NCard :bordered="false" size="small">
    <NCollapse>
      <NCollapseItem title="搜索条件" name="user-search">
        <NForm :model="model" label-placement="left" :label-width="80">
          <NGrid responsive="screen" item-responsive>
            <NFormItemGi span="24 s:12 m:6" label="用户名" path="user_name" class="pr-24px">
              <NInput v-model:value="model.user_name" placeholder="请输入用户名" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="手机号" path="phone" class="pr-24px">
              <NInput v-model:value="model.phone" placeholder="请输入手机号" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="邮箱" path="email" class="pr-24px">
              <NInput v-model:value="model.email" placeholder="请输入邮箱" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="状态" path="status" class="pr-24px">
              <NSelect
                v-model:value="model.status"
                :options="[
                  { label: '正常', value: 1 },
                  { label: '禁用', value: 0 }
                ]"
                placeholder="请选择状态"
                clearable
              />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="最小余额" path="balance_min" class="pr-24px">
              <NInputNumber v-model:value="model.balance_min" placeholder="最小余额" :precision="2" :min="0" />
            </NFormItemGi>
            <NFormItemGi span="24 s:12 m:6" label="最大余额" path="balance_max" class="pr-24px">
              <NInputNumber v-model:value="model.balance_max" placeholder="最大余额" :precision="2" :min="0" />
            </NFormItemGi>
            <NFormItemGi span="24 m:12" class="pr-24px">
              <NSpace class="w-full" justify="end">
                <NButton @click="reset">重置</NButton>
                <NButton type="primary" ghost @click="search">搜索</NButton>
              </NSpace>
            </NFormItemGi>
          </NGrid>
        </NForm>
      </NCollapseItem>
    </NCollapse>
  </NCard>
</template>

<style scoped></style>
