<template>
  <NModal
    v-model:show="visible"
    preset="dialog"
    :title="formModel.id ? '编辑套餐' : '新增套餐'"
    :style="{ width: '600px' }"
    @close="handleClose"
  >
    <NForm
      ref="formRef"
      :model="formModel"
      :rules="rules"
      label-placement="left"
      label-width="auto"
      require-mark-placement="right-hanging"
    >
      <NFormItem label="套餐名称" path="name">
        <NInput 
          v-model:value="formModel.name" 
          placeholder="请输入套餐名称，如：100元、快充100、四川100电费" 
          :maxlength="50"
        />
        <template #feedback>
          <div class="text-gray-400 text-xs mt-1" style="color: red;">
            请自定义套餐名称，方便识别。建议包含面值数字，如：100元、快充100、四川100电费
          </div>
        </template>
      </NFormItem>
      <NFormItem label="产品ID" path="product_id">
        <NInput 
          v-model:value="formModel.product_id" 
          placeholder="请输入产品 ID"
          :min="1"
          :precision="0"
          :show-button="false"
        />
        <template #feedback>
          <div class="text-gray-400 text-xs mt-1" style="color: red;">
            产品ID、商品ID、编码等，一般在供应商开的后台都能找到
          </div>
        </template>
      </NFormItem>
      <NFormItem label="面值" path="par_value">
        <NInputNumber v-model:value="formModel.par_value" placeholder="请输入面值" />
      </NFormItem>
      <NFormItem label="成本价格" path="price">
        <NInputNumber v-model:value="formModel.price" placeholder="请输入价格" />
      </NFormItem>
      <NFormItem label="允许省份" path="allow_provinces">
        <NInput v-model:value="formModel.allow_provinces" placeholder="请输入允许省份" />
      </NFormItem>
      <NFormItem label="禁止省份" path="forbid_provinces">
        <NInput v-model:value="formModel.forbid_provinces" placeholder="请输入禁止省份" />
      </NFormItem>
      <NFormItem label="允许城市" path="allow_cities">
        <NInput v-model:value="formModel.allow_cities" placeholder="请输入允许城市" />
      </NFormItem>
      <NFormItem label="禁止城市" path="forbid_cities">
        <NInput v-model:value="formModel.forbid_cities" placeholder="请输入禁止城市" />
      </NFormItem>
      <NFormItem label="状态" path="status">
        <NSwitch v-model:value="formModel.status" :checked-value="1" :unchecked-value="0" />
        <template #feedback>
          <div class="text-gray-400 text-xs mt-1">
            <NTag type="success" size="small">启用</NTag> 或 <NTag type="error" size="small">禁用</NTag>
          </div>
        </template>
      </NFormItem>
    </NForm>
    <template #action>
      <NSpace>
        <NButton @click="hideModal">取消</NButton>
        <NButton type="primary" @click="onSubmit">确定</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useForm } from '@/hooks/useForm';
import { useMessage } from 'naive-ui';
import { request } from '@/service/request';
import { NButton, NCard, NForm, NFormItem, NSpace, NInput, NSelect, NSwitch, NModal, NTag } from 'naive-ui';

interface PlatformAPIParam {
  id?: number;
  api_id: number;
  name: string;
  product_id: string;
  par_value: number;
  price: number;
  allow_provinces: string;
  forbid_provinces: string;
  allow_cities: string;
  forbid_cities: string;
  status: number;
}

const message = useMessage();
const visible = ref(false);
const emit = defineEmits(['success']);

const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();

// 设置验证规则
rules.value = {
  name: {
    required: true,
    message: '请输入套餐名称',
    trigger: ['blur', 'change']
  },
  product_id: {
    required: true,
    message: '请输入有效的产品ID1',
    trigger: ['blur', 'change'],
  },
  par_value: {
    required: true,
    message: '请输入面值',
    trigger: ['blur', 'change'],
    type: 'number',
  },
  price: {
    required: true,
    message: '请输入有成本价格',
    trigger: ['blur', 'change'],
    type: 'number',
  }
};

// 监听对话框显示状态
watch(visible, (newVal) => {
  if (newVal && !formModel.value.id) {
    // 新增时重置表单
    console.log("ffffffff");
    // resetForm();
  }
});

// 新增参数
const add = (apiId: number) => {
  // resetForm();
  
  formModel.value.api_id = apiId;
  console.log(formModel.value,"gggg",apiId);
  visible.value = true;
};

// 编辑参数
const edit = (row: PlatformAPIParam) => {
  formModel.value = { ...row };
  visible.value = true;
};

// 隐藏弹窗
const hideModal = () => {
  visible.value = false;
};

// 关闭对话框时重置表单
const handleClose = () => {
  // resetForm();
  hideModal();
};

// 提交表单
const onSubmit = async () => {
  try {
    await handleSubmit();
    if (formModel.value.id) {
      await request({
        url: `/platform/api/params/${formModel.value.id}`,
        method: 'PUT',
        data: formModel.value
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/platform/api/params',
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

defineExpose({
  add,
  edit
});
</script> 