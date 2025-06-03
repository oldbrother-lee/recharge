<template>
  <el-form ref="form" :model="form" :rules="rules" label-width="100px">
    <el-form-item label="商品名称" prop="name">
      <el-input v-model="form.name" placeholder="请输入商品名称" />
    </el-form-item>
    <el-form-item label="商品描述" prop="description">
      <el-input v-model="form.description" type="textarea" placeholder="请输入商品描述" />
    </el-form-item>
    <el-form-item label="商品类型" prop="type">
      <el-select v-model="form.type" placeholder="请选择商品类型">
        <el-option label="话费充值" :value="1" />
        <el-option label="流量充值" :value="2" />
      </el-select>
    </el-form-item>
    <el-form-item label="商品分类" prop="category_id">
      <el-select v-model="form.category_id" placeholder="请选择商品分类">
        <el-option
          v-for="item in categoryOptions"
          :key="item.id"
          :label="item.name"
          :value="item.id"
        />
      </el-select>
    </el-form-item>
    <el-form-item label="运营商" prop="isp">
      <el-select v-model="form.isp" placeholder="请选择运营商">
        <el-option label="移动" value="移动" />
        <el-option label="联通" value="联通" />
        <el-option label="电信" value="电信" />
      </el-select>
    </el-form-item>
    <el-form-item label="价格" prop="price">
      <el-input-number v-model="form.price" :precision="2" :step="0.1" :min="0" />
    </el-form-item>
    <el-form-item label="最大价格" prop="max_price">
      <el-input-number v-model="form.max_price" :precision="2" :step="0.1" :min="0" />
    </el-form-item>
    <el-form-item label="代金券价格" prop="voucher_price">
      <el-input v-model="form.voucher_price" placeholder="请输入代金券价格" />
    </el-form-item>
    <el-form-item label="代金券名称" prop="voucher_name">
      <el-input v-model="form.voucher_name" placeholder="请输入代金券名称" />
    </el-form-item>
    <el-form-item label="显示样式" prop="show_style">
      <el-select v-model="form.show_style" placeholder="请选择显示样式">
        <el-option label="默认" :value="1" />
        <el-option label="特殊" :value="2" />
      </el-select>
    </el-form-item>
    <el-form-item label="API失败样式" prop="api_fail_style">
      <el-select v-model="form.api_fail_style" placeholder="请选择API失败样式">
        <el-option label="默认" :value="1" />
        <el-option label="特殊" :value="2" />
      </el-select>
    </el-form-item>
    <el-form-item label="允许省份" prop="allow_provinces">
      <el-input v-model="form.allow_provinces" placeholder="请输入允许省份，多个用逗号分隔" />
    </el-form-item>
    <el-form-item label="允许城市" prop="allow_cities">
      <el-input v-model="form.allow_cities" placeholder="请输入允许城市，多个用逗号分隔" />
    </el-form-item>
    <el-form-item label="禁止省份" prop="forbid_provinces">
      <el-input v-model="form.forbid_provinces" placeholder="请输入禁止省份，多个用逗号分隔" />
    </el-form-item>
    <el-form-item label="禁止城市" prop="forbid_cities">
      <el-input v-model="form.forbid_cities" placeholder="请输入禁止城市，多个用逗号分隔" />
    </el-form-item>
    <el-form-item label="API延迟" prop="api_delay">
      <el-input v-model="form.api_delay" placeholder="请输入API延迟" />
    </el-form-item>
    <el-form-item label="排序" prop="sort">
      <el-input-number v-model="form.sort" :min="0" />
    </el-form-item>
    <el-form-item label="状态" prop="status">
      <el-radio-group v-model="form.status">
        <el-radio :label="1">启用</el-radio>
        <el-radio :label="0">禁用</el-radio>
      </el-radio-group>
    </el-form-item>
    <el-form-item label="API启用" prop="api_enabled">
      <el-switch v-model="form.api_enabled" />
    </el-form-item>
    <el-form-item label="是否解码" prop="is_decode">
      <el-switch v-model="form.is_decode" />
    </el-form-item>
    <el-form-item label="备注" prop="remark">
      <el-input v-model="form.remark" type="textarea" placeholder="请输入备注" />
    </el-form-item>
    <el-form-item>
      <el-button type="primary" @click="submitForm">确定</el-button>
      <el-button @click="$emit('cancel')">取消</el-button>
    </el-form-item>
  </el-form>
</template>

<script lang="ts" setup>
import { ref, reactive, watch } from 'vue'
import { createProduct, updateProduct } from '@/service/api/product'
import type { FormInstance } from 'element-plus'

const props = defineProps<{
  formData: Record<string, any>
  categoryOptions: Array<Record<string, any>>
}>()

const emit = defineEmits<{
  (e: 'success'): void
  (e: 'cancel'): void
}>()

const formRef = ref<FormInstance>()

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
})

const rules = {
  name: [{ required: true, message: '请输入商品名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择商品类型', trigger: 'change' }],
  category_id: [{ required: true, message: '请选择商品分类', trigger: 'change' }],
  price: [{ required: true, message: '请输入价格', trigger: 'blur' }]
}

watch(() => props.formData, (val) => {
  if (val) {
    Object.assign(form, val)
  }
}, { immediate: true })

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      const isEdit = !!form.id
      const request = isEdit ? updateProduct : createProduct
      await request(form)
      window.$message.success(`${isEdit ? '修改' : '新增'}成功`)
      emit('success')
    }
  })
}
</script> 