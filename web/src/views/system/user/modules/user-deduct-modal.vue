<template>
  <n-modal v-model:show="visible" preset="dialog" title="余额扣款" :style="{ width: '500px' }">
    <n-form
      ref="formRef"
      :model="formModel"
      :rules="rules"
      label-placement="left"
      label-width="auto"
      require-mark-placement="right-hanging"
    >
      <n-form-item label="用户名">
        <span>{{ user?.username }}</span>
      </n-form-item>
      <n-form-item label="当前余额">
        <span>¥{{ user?.balance?.toFixed(2) || '0.00' }}</span>
      </n-form-item>
      <n-form-item label="扣款金额" path="amount">
        <n-input-number
          v-model:value="formModel.amount"
          :min="0.01"
          :precision="2"
          placeholder="请输入扣款金额"
        />
      </n-form-item>
      <n-form-item label="扣款类型" path="style">
        <n-select
          v-model:value="formModel.style"
          :options="[
            { label: '订单扣款', value: 1 },
            { label: '手动扣款', value: 2 }
          ]"
          placeholder="请选择扣款类型"
        />
      </n-form-item>
      <n-form-item label="备注" path="remark">
        <n-input
          v-model:value="formModel.remark"
          type="textarea"
          placeholder="请输入备注信息"
        />
      </n-form-item>
    </n-form>
    <template #action>
      <n-space>
        <n-button @click="closeModal">取消</n-button>
        <n-button type="primary" :loading="loading" @click="handleSubmit">确定</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { request } from '@/service/request';

const props = defineProps<{
  visible: boolean;
  user?: any;
}>();
const emit = defineEmits(['update:visible', 'submitted']);

const visible = ref(props.visible);
watch(() => props.visible, v => (visible.value = v));
watch(visible, v => emit('update:visible', v));

const user = ref(props.user);
watch(() => props.user, v => (user.value = v));

const formRef = ref();
const formModel = ref({
  amount: 0,
  style: null,
  remark: ''
});
const rules = {
  amount: [
    { required: true, message: '请输入扣款金额', trigger: 'blur',type: 'number' },
    { validator: (rule: any, value: any) => {
        if (typeof value !== 'number' || isNaN(value) || value <= 0) {
          return new Error('请输入有效的扣款金额');
        }
        return true;
      }, trigger: 'blur' }
  ],
  style: [
    { required: true, message: '请选择扣款类型', trigger: 'change',type: 'number' }
  ]
};
const loading = ref(false);

function closeModal() {
  visible.value = false;
}

async function handleSubmit() {
  await formRef.value?.validate();
  loading.value = true;
  try {
    await request({
      url: '/balance/deduct',
      method: 'POST',
      data: {
        user_id: user.value.id,
        amount: formModel.value.amount,
        style: formModel.value.style,
        remark: formModel.value.remark
      }
    });
    closeModal();
    emit('submitted');
  } finally {
    loading.value = false;
  }
}
</script> 