<template>
  <n-modal v-model:show="visible" preset="dialog" title="授信设置" :style="{ width: '500px' }">
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
      <n-form-item label="当前授信">
        <span>¥{{ user?.credit?.toFixed(2) || '0.00' }}</span>
      </n-form-item>
      <n-form-item label="授信额度" path="creditLimit">
        <n-input-number
          v-model:value="formModel.creditLimit"
          :min="0"
          :precision="2"
          placeholder="请输入授信额度"
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
  creditLimit: 0,
  remark: ''
});
const rules = {
  creditLimit: [
    { required: true, message: '请输入授信额度', trigger: 'blur',type: 'number' },
    { validator: (rule: any, value: any) => {
        if (typeof value !== 'number' || isNaN(value) || value < 0) {
          return new Error('请输入有效的授信额度');
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
      url: '/credit/set',
      method: 'POST',
      data: {
        user_id: user.value.id,
        creditLimit: formModel.value.creditLimit,
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