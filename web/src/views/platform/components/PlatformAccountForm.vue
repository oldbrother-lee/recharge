<template>
    <NModal
      v-model:show="visible"
      preset="dialog"
      :title="formModel.id ? '编辑平台账号' : '新增平台账号'"
      :style="{ width: '600px' }"
    >
      <NForm
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <NFormItem label="平台" path="platform_id">
          <NSelect
            v-model:value="formModel.platform_id"
            :options="platformOptions"
            placeholder="请选择平台"
            :disabled="!!formModel.id"
          />
        </NFormItem>
        <NFormItem label="账号名称" path="name">
          <NInput v-model:value="formModel.account_name" placeholder="请输入账号名称" />
        </NFormItem>
        <NFormItem label="账号类型" path="type">
          <NSelect
            v-model:value="formModel.type"
            :options="[
              { label: '测试账号', value: 1 },
              { label: '正式账号', value: 2 }
            ]"
            placeholder="请选择账号类型"
          />
        </NFormItem>
        <NFormItem label="AppKey" path="app_key">
          <NInput v-model:value="formModel.app_key" placeholder="请输入AppKey" />
        </NFormItem>
        <NFormItem label="AppSecret" path="app_secret">
          <NInput v-model:value="formModel.app_secret" placeholder="请输入AppSecret" />
        </NFormItem>
        <NFormItem label="描述" path="description">
          <NInput v-model:value="formModel.description" type="textarea" placeholder="请输入描述" />
        </NFormItem>
        <NFormItem label="状态" path="status">
          <NSwitch v-model:value="formModel.status" :checked-value="1" :unchecked-value="0" />
        </NFormItem>
        <NFormItem label="推单状态" path="push_status">
          <NSelect
            v-model:value="formModel.push_status"
            :options="[
              { label: '开启', value: 1 },
              { label: '关闭', value: 2 }
            ]"
            placeholder="请选择推单状态"
          />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="hideModal">取消</NButton>
          <NButton type="primary" @click="handleFormSubmit">确定</NButton>
        </NSpace>
      </template>
    </NModal>
  </template>
  
  <script setup lang="ts">
  import { ref, onMounted, computed } from 'vue';
  import { useModal } from '@/hooks/useModal';
  import { useForm } from '@/hooks/useForm';
  import { useMessage } from 'naive-ui';
  import { request } from '@/service/request';
  import { NForm, NFormItem, NInput, NSelect, NSwitch, NButton, NSpace } from 'naive-ui';
  import type { FormRules } from 'naive-ui';
  
  interface PlatformAccount {
    id?: number;
    platform_id: number | null;
    account_name: string;
    type: number;
    app_key: string;
    app_secret: string;
    description: string;
    status: number;
    push_status: number;
  }
  
  interface Platform {
    id: number;
    name: string;
  }
  
  const message = useMessage();
  const { visible, showModal, hideModal } = useModal();
  const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();
  
  // 平台选项
  const platformOptions = ref<{ label: string; value: number }[]>([]);
  
  // 获取平台列表
  const fetchPlatforms = async () => {
    try {
      const res = await request({
        url: '/platform/list',
        method: 'GET',
        params: {
          page: 1,
          page_size: 100
        }
      });
      if (res.data) {
        platformOptions.value = res.data.list.map((item: Platform) => ({
          label: item.name,
          value: item.id
        }));
      }
    } catch (error) {
      message.error('获取平台列表失败');
    }
  };
  
  // 提交表单
  const handleFormSubmit = async () => {
    try {
      await handleSubmit();
      if (formModel.value.id) {
        await request({
          url: `/platform/account/${formModel.value.id}`,
          method: 'PUT',
          data: formModel.value
        });
        message.success('更新成功');
      } else {
        await request({
          url: '/platform/account',
          method: 'POST',
          data: formModel.value
        });
        message.success('创建成功');
      }
      hideModal();
      emit('success');
    } catch (error) {
      message.error('操作失败');
    }
  };
  
  // 重置表单
  const reset = () => {
    resetForm();
    formModel.value = {
      platform_id: null,
      account_name: '',
      type: 1,
      app_key: '',
      app_secret: '',
      description: '',
      status: 1,
      push_status: 1
    };
  };
  
  // 编辑账号
  const edit = (row: PlatformAccount) => {
    formModel.value = { ...row };
    showModal();
  };
  
  // 新增账号
  const add = (platformId?: number) => {
    reset();
    if (platformId) {
      formModel.value.platform_id = platformId;
    }
    showModal();
  };
  
  // 暴露方法
  defineExpose({
    edit,
    add
  });
  
  // 定义事件
  const emit = defineEmits(['success']);
  
  onMounted(() => {
    fetchPlatforms();
  });
  </script>