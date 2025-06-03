<template>
  <n-modal v-model:show="visible" preset="dialog" title="余额充值" :style="{ width: '500px' }">
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
      <n-form-item label="充值金额" path="amount">
        <n-input-number
          v-model:value="formModel.amount"
          :min="0.01"
          :precision="2"
          placeholder="请输入充值金额"
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
  remark: ''
});

watch([
  () => props.visible,
  () => props.user
], ([visible, user]) => {
  if (visible && user) {
    formModel.value = {
      amount: 0,
      remark: ''
    };
  }
});

const rules = {
  amount: [
    { required: true, message: '请输入充值金额', trigger: 'blur',type: 'number' },
    { validator: (rule: any, value: any) => {
        const num = Number(value);
        if (isNaN(num) || num <= 0) {
          return new Error('请输入有效的充值金额');
        }
        return true;
      }, trigger: 'blur' }
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
      url: '/balance/recharge',
      method: 'POST',
      data: {
        user_id: user.value.id,
        amount: formModel.value.amount,
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