<script lang="ts" setup>
import { reactive, ref, watch } from 'vue';
import type { FormInstance } from 'element-plus';
import { createProduct, updateProduct } from '@/service/api/product';

const props = defineProps<{
  formData: Record<string, any>;
  categoryOptions: Array<Record<string, any>>;
}>();

const emit = defineEmits<{
  (e: 'success'): void;
  (e: 'cancel'): void;
}>();

const formRef = ref<FormInstance>();

const form = reactive({
  name: '',
  description: '',
  type: 1,
  category_id: undefined,
  isp: '',
  price: 0,
  max_price: 0,
  voucher_price: '',
  voucher_name: '',
  show_style: 1,
  api_fail_style: 1,
  allow_provinces: '',
  allow_cities: '',
  forbid_provinces: '',
  forbid_cities: '',
  api_delay: '',
  sort: 0,
  status: 1,
  api_enabled: false,
  is_decode: false,
  remark: ''
});

const rules = {
  name: [{ required: true, message: '请输入商品名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择商品类型', trigger: 'change' }],
  category_id: [{ required: true, message: '请选择商品分类', trigger: 'change' }],
  price: [{ required: true, message: '请输入价格', trigger: 'blur' }]
};

watch(
  () => props.formData,
  val => {
    if (val) {
      Object.assign(form, val);
    }
  },
  { immediate: true }
);

const submitForm = async () => {
  if (!formRef.value) return;
  await formRef.value.validate(async valid => {
    if (valid) {
      const isEdit = Boolean(form.id);
      const request = isEdit ? updateProduct : createProduct;
      await request(form);
      window.$message.success(`${isEdit ? '修改' : '新增'}成功`);
      emit('success');
    }
  });
};
</script>

<template>
  <ElForm ref="form" :model="form" :rules="rules" label-width="100px">
    <ElFormItem label="商品名称" prop="name">
      <ElInput v-model="form.name" placeholder="请输入商品名称" />
    </ElFormItem>
    <ElFormItem label="商品描述" prop="description">
      <ElInput v-model="form.description" type="textarea" placeholder="请输入商品描述" />
    </ElFormItem>
    <ElFormItem label="商品类型" prop="type">
      <ElSelect v-model="form.type" placeholder="请选择商品类型">
        <ElOption label="话费充值" :value="1" />
        <ElOption label="流量充值" :value="2" />
      </ElSelect>
    </ElFormItem>
    <ElFormItem label="商品分类" prop="category_id">
      <ElSelect v-model="form.category_id" placeholder="请选择商品分类">
        <ElOption v-for="item in categoryOptions" :key="item.id" :label="item.name" :value="item.id" />
      </ElSelect>
    </ElFormItem>
    <ElFormItem label="运营商" prop="isp">
      <ElSelect v-model="form.isp" placeholder="请选择运营商">
        <ElOption label="移动" value="移动" />
        <ElOption label="联通" value="联通" />
        <ElOption label="电信" value="电信" />
      </ElSelect>
    </ElFormItem>
    <ElFormItem label="价格" prop="price">
      <ElInputNumber v-model="form.price" :precision="2" :step="0.1" :min="0" />
    </ElFormItem>
    <ElFormItem label="最大价格" prop="max_price">
      <ElInputNumber v-model="form.max_price" :precision="2" :step="0.1" :min="0" />
    </ElFormItem>
    <ElFormItem label="代金券价格" prop="voucher_price">
      <ElInput v-model="form.voucher_price" placeholder="请输入代金券价格" />
    </ElFormItem>
    <ElFormItem label="代金券名称" prop="voucher_name">
      <ElInput v-model="form.voucher_name" placeholder="请输入代金券名称" />
    </ElFormItem>
    <ElFormItem label="显示样式" prop="show_style">
      <ElSelect v-model="form.show_style" placeholder="请选择显示样式">
        <ElOption label="默认" :value="1" />
        <ElOption label="特殊" :value="2" />
      </ElSelect>
    </ElFormItem>
    <ElFormItem label="API失败样式" prop="api_fail_style">
      <ElSelect v-model="form.api_fail_style" placeholder="请选择API失败样式">
        <ElOption label="默认" :value="1" />
        <ElOption label="特殊" :value="2" />
      </ElSelect>
    </ElFormItem>
    <ElFormItem label="允许省份" prop="allow_provinces">
      <ElInput v-model="form.allow_provinces" placeholder="请输入允许省份，多个用逗号分隔" />
    </ElFormItem>
    <ElFormItem label="允许城市" prop="allow_cities">
      <ElInput v-model="form.allow_cities" placeholder="请输入允许城市，多个用逗号分隔" />
    </ElFormItem>
    <ElFormItem label="禁止省份" prop="forbid_provinces">
      <ElInput v-model="form.forbid_provinces" placeholder="请输入禁止省份，多个用逗号分隔" />
    </ElFormItem>
    <ElFormItem label="禁止城市" prop="forbid_cities">
      <ElInput v-model="form.forbid_cities" placeholder="请输入禁止城市，多个用逗号分隔" />
    </ElFormItem>
    <ElFormItem label="API延迟" prop="api_delay">
      <ElInput v-model="form.api_delay" placeholder="请输入API延迟" />
    </ElFormItem>
    <ElFormItem label="排序" prop="sort">
      <ElInputNumber v-model="form.sort" :min="0" />
    </ElFormItem>
    <ElFormItem label="状态" prop="status">
      <ElRadioGroup v-model="form.status">
        <ElRadio :label="1">启用</ElRadio>
        <ElRadio :label="0">禁用</ElRadio>
      </ElRadioGroup>
    </ElFormItem>
    <ElFormItem label="API启用" prop="api_enabled">
      <ElSwitch v-model="form.api_enabled" />
    </ElFormItem>
    <ElFormItem label="是否解码" prop="is_decode">
      <ElSwitch v-model="form.is_decode" />
    </ElFormItem>
    <ElFormItem label="备注" prop="remark">
      <ElInput v-model="form.remark" type="textarea" placeholder="请输入备注" />
    </ElFormItem>
    <ElFormItem>
      <ElButton type="primary" @click="submitForm">确定</ElButton>
      <ElButton @click="$emit('cancel')">取消</ElButton>
    </ElFormItem>
  </ElForm>
</template>
