<template>
  <n-drawer v-model:show="visible" :width="360" placement="right">
    <n-drawer-content :title="operateType === 'add' ? '新增用户' : '编辑用户'">
      <n-form
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="formModel.username" placeholder="请输入用户名" />
        </n-form-item>
        <n-form-item label="密码" path="password" v-if="operateType === 'add'">
          <n-input
            v-model:value="formModel.password"
            type="password"
            placeholder="请输入密码"
          />
        </n-form-item>
        <!-- <n-form-item label="昵称" path="nickname">
          <n-input v-model:value="formModel.nickname" placeholder="请输入昵称" />
        </n-form-item>
        <n-form-item label="邮箱" path="email">
          <n-input v-model:value="formModel.email" placeholder="请输入邮箱" />
        </n-form-item> -->
        <n-form-item label="手机号" path="phone">
          <n-input v-model:value="formModel.phone" placeholder="请输入手机号" />
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-select
            v-model:value="formModel.status"
            :options="[
              { label: '正常', value: 1 },
              { label: '禁用', value: 0 }
            ]"
            placeholder="请选择状态"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space>
          <n-button @click="closeDrawer">取消</n-button>
          <n-button type="primary" @click="handleSubmit">确定</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { request } from '@/service/request';
import { useMessage } from 'naive-ui';

const props = defineProps<{
  visible: boolean;
  operateType: 'add' | 'edit';
  rowData?: any;
}>();
const emit = defineEmits(['update:visible', 'submitted']);

const visible = ref(props.visible);
watch(() => props.visible, v => (visible.value = v));
watch(visible, v => emit('update:visible', v));

const operateType = ref(props.operateType);
watch(() => props.operateType, v => (operateType.value = v));

const formRef = ref();
const formModel = ref({
  id: undefined,
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  status: 1
});
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  // nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  // email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }],
  phone: [{ required: true, message: '请输入手机号', trigger: 'blur' }],
  status: [{ required: true, type: 'number' as const, message: '请选择状态', trigger: 'change' }]
};

watch(
  () => props.rowData,
  (val) => {
    if (operateType.value === 'edit' && val) {
      formModel.value = { ...val };
      formModel.value.password = '';
    } else {
      formModel.value = {
        id: undefined,
        username: '',
        password: '',
        nickname: '',
        email: '',
        phone: '',
        status: 1
      };
    }
  },
  { immediate: true }
);

const message = useMessage();

function closeDrawer() {
  visible.value = false;
}

async function handleSubmit() {
  await formRef.value?.validate();
  try {
    if (operateType.value === 'edit' && formModel.value.id) {
      // 编辑
      const updateData = { ...formModel.value };
      delete (updateData as any).password;
      await request({
        url: `/users/${formModel.value.id}`,
        method: 'PUT',
        data: updateData
      });
      message.success('编辑成功');
    } else {
      // 新增
      await request({
        url: '/user/register',
        method: 'POST',
        data: formModel.value
      });
      message.success('新增成功');
    }
    closeDrawer();
    emit('submitted');
  } catch (e) {
    message.error((e as any)?.message || '操作失败');
  }
}
</script> 