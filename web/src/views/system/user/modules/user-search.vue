<template>
  <n-card :bordered="false" size="small">
    <n-collapse>
      <n-collapse-item title="搜索条件" name="user-search">
        <n-form :model="model" label-placement="left" :label-width="80">
          <n-grid responsive="screen" item-responsive>
            <n-form-item-gi span="24 s:12 m:6" label="用户名" path="user_name" class="pr-24px">
              <n-input v-model:value="model.user_name" placeholder="请输入用户名" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="手机号" path="phone" class="pr-24px">
              <n-input v-model:value="model.phone" placeholder="请输入手机号" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="邮箱" path="email" class="pr-24px">
              <n-input v-model:value="model.email" placeholder="请输入邮箱" />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="状态" path="status" class="pr-24px">
              <n-select
                v-model:value="model.status"
                :options="[
                  { label: '正常', value: 1 },
                  { label: '禁用', value: 0 }
                ]"
                placeholder="请选择状态"
                clearable
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="最小余额" path="balance_min" class="pr-24px">
              <n-input-number
                v-model:value="model.balance_min"
                placeholder="最小余额"
                :precision="2"
                :min="0"
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 s:12 m:6" label="最大余额" path="balance_max" class="pr-24px">
              <n-input-number
                v-model:value="model.balance_max"
                placeholder="最大余额"
                :precision="2"
                :min="0"
              />
            </n-form-item-gi>
            <n-form-item-gi span="24 m:12" class="pr-24px">
              <n-space class="w-full" justify="end">
                <n-button @click="reset">
                  重置
                </n-button>
                <n-button type="primary" ghost @click="search">
                  搜索
                </n-button>
              </n-space>
            </n-form-item-gi>
          </n-grid>
        </n-form>
      </n-collapse-item>
    </n-collapse>
  </n-card>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { NCard, NCollapse, NCollapseItem, NForm, NGrid, NFormItemGi, NInput, NSelect, NInputNumber, NSpace, NButton } from 'naive-ui';

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

<style scoped>

</style> 