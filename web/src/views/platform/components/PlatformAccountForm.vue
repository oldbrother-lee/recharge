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
        <NFormItem label="登录密码" path="account_password">
          <NInput v-model:value="formModel.account_password" type="password" show-password-on="click" placeholder="请输入登录密码" />
        </NFormItem>
        <NFormItem label="描述" path="description">
          <NInput v-model:value="formModel.description" type="textarea" placeholder="请输入描述" />
        </NFormItem>
        <NFormItem label="状态" path="status">
          <NSwitch v-model:value="formModel.status" :checked-value="1" :unchecked-value="0" />
        </NFormItem>
        <NFormItem label="模式" path="push_status">
          <NRadioGroup
            v-model:value="formModel.push_status"
            @update:value="(v:number) => { formModel.enable_pull_order = v === 1 ? false : true }"
          >
            <NRadio :value="1">推单模式</NRadio>
            <NRadio :value="0">拉单模式</NRadio>
          </NRadioGroup>
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
  import { ref, onMounted, computed, watch } from 'vue';
  import { useModal } from '@/hooks/useModal';
  import { useForm } from '@/hooks/useForm';
  import { useMessage } from 'naive-ui';
  import { request } from '@/service/request';
  import { NForm, NFormItem, NInput, NSelect, NSwitch, NButton, NSpace, NRadioGroup, NRadio } from 'naive-ui';
  import type { FormRules } from 'naive-ui';
  
  interface PlatformAccount {
    id?: number;
    platform_id: number | null;
    account_name: string;
    type: number;
    app_key: string;
    app_secret: string;
    account_password: string;
    description: string;
    status: number;
    push_status: number;
    enable_pull_order?: boolean;
    max_concurrency?: number;
    poll_interval_sec?: number;
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
      account_password: '',
      description: '',
      status: 1,
      push_status: 1,
      enable_pull_order: false,
      max_concurrency: 1,
      poll_interval_sec: 10
    };
  };
  
  // 编辑账号
  const edit = (row: PlatformAccount) => {
    formModel.value = { ...row };
    // 同步模式映射：enable_pull_order=true 则显示拉单模式（push_status=0）
    if (formModel.value.enable_pull_order === true) {
      formModel.value.push_status = 0;
    } else if (formModel.value.push_status !== 1) {
      // 默认推单模式
      formModel.value.push_status = 1;
      formModel.value.enable_pull_order = false;
    }
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