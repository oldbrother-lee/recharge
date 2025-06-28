<template>
  <NModal v-model:show="visible" preset="dialog" :title="modalTitle" style="width: 900px; max-width: 90vw;">
    <div style="max-height: 600px; overflow-y: auto; padding: 16px;">
      <!-- 基本信息 -->
      <NGrid :cols="24" :x-gap="16" class="mb-6">
        <NGridItem :span="6">
          <div class="text-center">
            <div class="text-sm text-gray-500 mb-1">商品ID</div>
            <div class="text-base font-medium">{{ formData.goods_id || '10000001' }}</div>
          </div>
        </NGridItem>
        <NGridItem :span="6">
          <div class="text-center">
            <div class="text-sm text-gray-500 mb-1">渠道名称</div>
            <div class="text-base font-medium">推单用户测试渠道</div>
          </div>
        </NGridItem>
        <NGridItem :span="6">
          <div class="text-center">
            <div class="text-sm text-gray-500 mb-1">商品名称</div>
            <div class="text-base font-medium">{{ productName || '话费充值-移动' }}</div>
          </div>
        </NGridItem>
        <NGridItem :span="6">
          <div class="text-center">
            <div class="text-sm text-gray-500 mb-1">面值规格</div>
            <div class="text-base font-medium">10</div>
          </div>
        </NGridItem>
      </NGrid>

      <NForm
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="120px"
      >
        <!-- 限制省份 -->
        <div class="mb-6">
          <div class="flex items-center mb-4">
            <span class="text-red-500 mr-1">*</span>
            <span class="text-base font-medium">限制省份:</span>
            <NRadioGroup v-model:value="formData.prov_limit_type" class="ml-4">
              <NRadio :value="1" class="text-orange-500">
                <span class="text-orange-500">支持全国</span>
              </NRadio>
              <NRadio :value="2" class="ml-4">
                <span class="text-orange-500">限制省份</span>
              </NRadio>
            </NRadioGroup>
          </div>
          
          <!-- 省份选择区域 -->
          <div v-if="formData.prov_limit_type === 2" class="mt-4">
            <div class="text-sm text-gray-600 mb-3">请选择支持的省份:</div>
            <div class="grid grid-cols-4 gap-4">
              <div v-for="province in allProvinces" :key="province.code" class="flex items-center">
                <NCheckbox 
                  :checked="selectedProvinces.includes(province.code)"
                  @update:checked="(checked) => handleProvinceChange(province.code, checked)"
                >
                  {{ province.name }}
                </NCheckbox>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入报价 -->
        <div class="mb-6">
          <div class="flex items-center mb-2">
            <span class="text-red-500 mr-1">*</span>
            <span class="text-base font-medium">输入报价:</span>
          </div>
          
          <div class="mb-4 p-3 bg-blue-50 rounded-lg border border-blue-200">
            <div class="text-sm text-gray-700 mb-1">
              <span class="font-medium text-blue-700">参考价:</span> ¥{{ referencePrice.toFixed(3) }} ({{ referencePriceWan }}万分比)
            </div>
            <div class="text-xs text-gray-600">
              <span class="font-medium">报价区间:</span> {{ priceRange.min }}~{{ priceRange.max }}万分比
            </div>
            <div class="text-xs text-blue-600 mt-1">
              💡 万分比说明: 输入的数值将除以10000后与参考价相乘得到最终价格
            </div>
          </div>
          
          <!-- 当限制省份但未选择省份时显示提示 -->
          <div v-if="formData.prov_limit_type === 2 && selectedProvinces.length === 0" class="text-center py-8 text-gray-400">
            未选择省份
          </div>
          
          <!-- 正常输入报价区域 -->
          <div v-else>
            <!-- 当选择限制省份且已选择省份时，为每个省份单独设置价格 -->
            <template v-if="formData.prov_limit_type === 2 && selectedProvinces.length > 0">
              <div class="space-y-4">
                <div 
                  v-for="provinceCode in selectedProvinces" 
                  :key="provinceCode"
                  class="p-4 border border-gray-200 rounded-lg bg-white"
                >
                  <div class="flex items-center gap-4 mb-2 flex-wrap">
                    <span class="bg-green-100 text-green-800 px-3 py-1 rounded text-sm min-w-16 text-center flex-shrink-0">
                      {{ allProvinces.find(p => p.code === provinceCode)?.name }}
                    </span>
                    
                    <NInput
                      v-model:value="provinceQuotes[provinceCode]"
                      placeholder="请输入报价"
                      style="width: 200px; flex-shrink: 0;"
                    />
                    
                    <span class="bg-blue-100 text-blue-800 px-3 py-1 rounded text-sm flex-shrink-0">万分比</span>
                    
                    <NSwitch 
                      v-model:value="provinceStatus[provinceCode]"
                      :checked-value="true"
                      :unchecked-value="false"
                      class="flex-shrink-0"
                    >
                      <template #checked>启用</template>
                      <template #unchecked>禁用</template>
                    </NSwitch>
                  </div>
                  
                  <div class="text-sm text-gray-600 mt-2 pl-4">
                    报价计算: {{ getProvinceCalculatedPrice(provinceCode) }} 元 (根据您的报价自动计算)
                  </div>
                </div>
                
                <!-- 批量操作区域 -->
                <div class="mt-4 p-4 bg-gray-50 rounded-lg">
                  <div class="flex items-center gap-4 mb-2 flex-wrap">
                    <span class="text-sm text-gray-600 flex-shrink-0">批量填充价格:</span>
                    <NInput
                      v-model:value="batchPrice"
                      placeholder="请输入统一价格"
                      style="width: 150px; flex-shrink: 0;"
                    />
                    <span class="bg-blue-100 text-blue-800 px-2 py-1 rounded text-xs flex-shrink-0">万分比</span>
                    <NButton size="small" type="primary" @click="applyBatchPrice" class="flex-shrink-0">一键填充</NButton>
                  </div>
                  <div v-if="batchPrice" class="text-xs text-gray-500 mt-2 pl-4">
                     计算结果: {{ (referencePrice * parseFloat(batchPrice || '0') / 10000).toFixed(3) }} 元
                   </div>
                </div>
              </div>
            </template>
            
            <!-- 当选择全国时，显示全国标签 -->
            <template v-else>
              <div class="p-4 border border-gray-200 rounded-lg bg-white">
                <div class="flex items-center gap-4 mb-2 flex-wrap">
                  <span class="bg-blue-100 text-blue-800 px-3 py-1 rounded text-sm flex-shrink-0">全国</span>
                  <NInput
                    v-model:value="nationalPrice"
                    placeholder="请输入报价"
                    style="width: 200px; flex-shrink: 0;"
                  />
                  <span class="bg-blue-100 text-blue-800 px-3 py-1 rounded text-sm flex-shrink-0">万分比</span>
                </div>
                
                <div class="text-sm text-gray-600 mt-2 pl-4">
                  报价计算: {{ calculatedPrice }} 元 (根据您的报价自动计算)
                </div>
              </div>
            </template>
          </div>
        </div>
        
        <!-- 关联外部编码 -->
        <div class="mb-6">
          <div class="text-base font-medium mb-4">关联外部编码 (非必填):</div>
          
          <!-- 当限制省份但未选择省份时显示提示 -->
          <div v-if="formData.prov_limit_type === 2 && selectedProvinces.length === 0" class="text-center py-8 text-gray-400">
            未选择省份
          </div>
          
          <!-- 当选择限制省份且已选择省份时，为每个省份显示独立的外部编码输入框 -->
          <div v-else-if="formData.prov_limit_type === 2 && selectedProvinces.length > 0" class="space-y-4">
            <div 
              v-for="provinceCode in selectedProvinces" 
              :key="provinceCode"
              class="flex items-center gap-4 flex-wrap"
            >
              <span class="bg-green-100 text-green-800 px-3 py-1 rounded text-sm flex-shrink-0 min-w-16">
                {{ allProvinces.find(p => p.code === provinceCode)?.name }}
              </span>
              <NInput
                v-model:value="provinceExternalCodes[provinceCode]"
                placeholder="请输入编码"
                style="width: 300px; flex-shrink: 0;"
              />
            </div>
          </div>
          
          <!-- 当选择全国时，显示单个外部编码输入框 -->
          <div v-else class="flex items-center gap-4 flex-wrap">
            <span class="flex-shrink-0">输入外部编码</span>
            <NInput
              v-model:value="formData.external_code"
              placeholder="请输入编码"
              style="width: 300px; flex-shrink: 0;"
            />
          </div>
        </div>
      </NForm>
    </div>

    <template #action>
      <NSpace>
        <NButton @click="handleCancel">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmit">
          确定
        </NButton>
      </NSpace>
    </template>
  </NModal>

  <!-- 添加省份弹窗 -->
  <NModal v-model:show="addProvinceVisible" preset="dialog" title="添加省份" class="w-500px">
    <NCard :bordered="false" size="small">
      <NForm ref="provinceFormRef" :model="newProvince" label-placement="left" label-width="100px">
        <NGrid :cols="24" :x-gap="16" :y-gap="16">
          <NGridItem :span="24">
            <NFormItem label="省份" path="prov">
              <NSelect
                v-model:value="newProvince.prov"
                :options="availableProvinces"
                placeholder="请选择省份"
              />
            </NFormItem>
          </NGridItem>
          
          <NGridItem v-if="formData.user_quote_type === 2" :span="24">
            <NFormItem label="报价" path="user_quote_payment">
              <NInputNumber
                v-model:value="newProvince.user_quote_payment"
                :precision="2"
                :min="0"
                placeholder="请输入报价"
                class="w-full"
              >
                <template #suffix>元</template>
              </NInputNumber>
            </NFormItem>
          </NGridItem>
          
          <NGridItem v-if="formData.external_code_link_type === 2" :span="24">
            <NFormItem label="编码" path="external_code">
              <NInput v-model:value="newProvince.external_code" placeholder="请输入外部编码" />
            </NFormItem>
          </NGridItem>
          
          <NGridItem :span="24">
            <NFormItem label="状态" path="status">
              <NRadioGroup v-model:value="newProvince.status">
                <NRadio :value="1">启用</NRadio>
                <NRadio :value="0">禁用</NRadio>
              </NRadioGroup>
            </NFormItem>
          </NGridItem>
        </NGrid>
      </NForm>
    </NCard>
    
    <template #action>
      <NSpace>
        <NButton @click="addProvinceVisible = false">取消</NButton>
        <NButton type="primary" @click="handleConfirmAddProvince">确定</NButton>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { ref, computed, reactive, h, watch } from 'vue';
import type { DataTableColumns, FormInst } from 'naive-ui';
import { NButton, NInputNumber, NPopconfirm, NInput, NRadio, NRadioGroup, NCard, NGrid, NGridItem, NSpace, NSwitch, NCheckbox, NModal, NForm, NFormItem, NSelect, NDataTable } from 'naive-ui';
import { updateBeeProductPrice, updateBeeProductProvince, type BeeProduct, type BeeProvince, type BeeUpdatePriceRequest, type BeeUpdateProvinceRequest } from '@/api/bee-platform';

interface Emits {
  success: [];
}

const emit = defineEmits<Emits>();

const visible = ref(false);
const submitting = ref(false);
const editType = ref<'price' | 'province'>('price');
const accountId = ref<number>(0);
const productName = ref('');
const formRef = ref<FormInst>();
const addProvinceVisible = ref(false);
const provinceFormRef = ref<FormInst>();

// 表单数据
const formData = reactive<BeeUpdatePriceRequest>({
  goods_id: 0,
  status: 1,
  prov_limit_type: 1,
  user_quote_type: 1,
  external_code_link_type: 1,
  user_quote_payment: 0,
  external_code: '',
  prov_info: []
});

// 省份表单数据
const provinceFormData = reactive<BeeUpdateProvinceRequest>({
  goods_id: 0,
  provs: []
});

// 新增省份数据
const newProvince = reactive<BeeProvince>({
  prov: '',
  user_quote_payment: 0,
  external_code: '',
  status: 1
});

// 价格相关数据
const nationalPrice = ref('');
const referencePrice = ref(10.0000);
const referencePriceWan = computed(() => Math.round(referencePrice.value * 10000));

// 选中的省份列表
const selectedProvinces = ref<string[]>([]);

// 每个省份的报价
const provinceQuotes = ref<Record<string, string>>({});

// 每个省份的状态
const provinceStatus = ref<Record<string, boolean>>({});

// 每个省份的外部编码
const provinceExternalCodes = ref<Record<string, string>>({});

// 批量填充价格
const batchPrice = ref('');

// 价格范围
const priceRange = reactive({
  min: 9000,
  max: 10000
});

// 计算属性：根据输入的报价计算价格
const calculatedPrice = computed(() => {
  if (!nationalPrice.value) return '0.000';
  const price = parseFloat(nationalPrice.value) || 0;
  return (referencePrice.value * price / 10000).toFixed(3);
});

// 计算属性：为每个省份计算价格
const getProvinceCalculatedPrice = (provinceCode: string) => {
  const quote = provinceQuotes.value[provinceCode];
  if (!quote) return '0.000';
  const price = parseFloat(quote) || 0;
  return (referencePrice.value * price / 10000).toFixed(3);
};

// 处理省份选择变化
const handleProvinceChange = (provinceCode: string, checked: boolean) => {
  if (checked) {
    if (!selectedProvinces.value.includes(provinceCode)) {
      selectedProvinces.value.push(provinceCode);
      // 初始化该省份的报价、状态和外部编码
      provinceQuotes.value[provinceCode] = '';
      provinceStatus.value[provinceCode] = true;
      provinceExternalCodes.value[provinceCode] = '';
    }
  } else {
    const index = selectedProvinces.value.indexOf(provinceCode);
    if (index > -1) {
      selectedProvinces.value.splice(index, 1);
      // 删除该省份的报价、状态和外部编码数据
      delete provinceQuotes.value[provinceCode];
      delete provinceStatus.value[provinceCode];
      delete provinceExternalCodes.value[provinceCode];
    }
  }
};

// 批量填充价格
const applyBatchPrice = () => {
  if (!batchPrice.value) {
    window.$message?.warning('请输入批量价格');
    return;
  }
  
  selectedProvinces.value.forEach(provinceCode => {
    provinceQuotes.value[provinceCode] = batchPrice.value;
  });
  
  window.$message?.success('批量填充价格成功');
};

// 重置表单数据
const resetForm = () => {
  Object.assign(formData, {
    goods_id: 0,
    status: 1,
    prov_limit_type: 1,
    user_quote_type: 1,
    external_code_link_type: 1,
    user_quote_payment: 0,
    external_code: '',
    prov_info: []
  });
  nationalPrice.value = '';
  selectedProvinces.value = [];
  provinceQuotes.value = {};
  provinceStatus.value = {};
  batchPrice.value = '';
};

// 所有省份列表
const allProvinces = [
  { code: 'beijing', name: '北京' },
  { code: 'tianjin', name: '天津' },
  { code: 'hebei', name: '河北' },
  { code: 'shanxi', name: '山西' },
  { code: 'neimenggu', name: '内蒙古' },
  { code: 'liaoning', name: '辽宁' },
  { code: 'jilin', name: '吉林' },
  { code: 'heilongjiang', name: '黑龙江' },
  { code: 'shanghai', name: '上海' },
  { code: 'jiangsu', name: '江苏' },
  { code: 'zhejiang', name: '浙江' },
  { code: 'anhui', name: '安徽' },
  { code: 'fujian', name: '福建' },
  { code: 'jiangxi', name: '江西' },
  { code: 'shandong', name: '山东' },
  { code: 'henan', name: '河南' },
  { code: 'hubei', name: '湖北' },
  { code: 'hunan', name: '湖南' },
  { code: 'guangdong', name: '广东' },
  { code: 'guangxi', name: '广西' },
  { code: 'hainan', name: '海南' },
  { code: 'chongqing', name: '重庆' },
  { code: 'sichuan', name: '四川' },
  { code: 'guizhou', name: '贵州' },
  { code: 'yunnan', name: '云南' },
  { code: 'xizang', name: '西藏' },
  { code: 'shaanxi', name: '陕西' },
  { code: 'gansu', name: '甘肃' },
  { code: 'qinghai', name: '青海' },
  { code: 'ningxia', name: '宁夏' },
  { code: 'xinjiang', name: '新疆' }
];

// 可选省份列表
const availableProvinces = computed(() => {
  const usedProvinces = formData.prov_info.map(item => item.prov);
  return allProvinces
    .filter(province => !usedProvinces.includes(province.code))
    .map(province => ({ label: province.name, value: province.code }));
});

// 弹窗标题
const modalTitle = computed(() => {
  return editType.value === 'price' ? '省份管理' : '省份配置';
});

// 省份配置表格列
const provinceColumns: DataTableColumns<BeeProvince> = [
  {
    title: '省份',
    key: 'prov',
    render(row) {
      const province = allProvinces.find(p => p.code === row.prov);
      return province?.name || row.prov;
    }
  },
  {
    title: '报价',
    key: 'user_quote_payment',
    render(row, index) {
      if (formData.user_quote_type === 1) {
        return `¥${formData.user_quote_payment.toFixed(2)}`;
      }
      return h(NInputNumber, {
        value: row.user_quote_payment,
        precision: 2,
        min: 0,
        size: 'small',
        onUpdateValue: (value) => {
          formData.prov_info[index].user_quote_payment = value || 0;
        }
      });
    }
  },
  {
    title: '外部编码',
    key: 'external_code',
    render(row, index) {
      if (formData.external_code_link_type === 1) {
        return formData.external_code;
      }
      return h(NInput, {
        value: row.external_code,
        size: 'small',
        onUpdateValue: (value) => {
          formData.prov_info[index].external_code = value;
        }
      });
    }
  },
  {
    title: '状态',
    key: 'status',
    render(row, index) {
      return h(NRadioGroup, {
        value: row.status,
        onUpdateValue: (value) => {
          formData.prov_info[index].status = value;
        }
      }, {
        default: () => [
          h(NRadio, { value: 1 }, '启用'),
          h(NRadio, { value: 0 }, '禁用')
        ]
      });
    }
  },
  {
    title: '操作',
    key: 'actions',
    render(row, index) {
      return h(NPopconfirm, {
        onPositiveClick: () => handleRemoveProvince(index)
      }, {
        default: () => '确认删除？',
        trigger: () => h(NButton, { type: 'error', size: 'small' }, '删除')
      });
    }
  }
];

// 表单验证规则
const rules = {
  status: { required: true, message: '请选择商品状态' },
  prov_limit_type: { required: true, message: '请选择省份限制类型' },
  user_quote_type: { required: true, message: '请选择报价类型' },
  external_code_link_type: { required: true, message: '请选择编码类型' },
  user_quote_payment: { required: true, message: '请输入报价金额' },
  external_code: { required: true, message: '请输入外部编码' }
};

// 添加省份
const handleAddProvince = () => {
  addProvinceVisible.value = true;
  Object.assign(newProvince, {
    prov: '',
    user_quote_payment: 0,
    external_code: '',
    status: 1
  });
};

// 确认添加省份
const handleConfirmAddProvince = async () => {
  if (!provinceFormRef.value) return;
  
  try {
    await provinceFormRef.value.validate();
    formData.prov_info.push({ ...newProvince });
    addProvinceVisible.value = false;
    window.$message?.success('添加省份成功');
  } catch (error) {
    console.error('添加省份失败:', error);
  }
};

// 删除省份
const handleRemoveProvince = (index: number) => {
  formData.prov_info.splice(index, 1);
  window.$message?.success('删除省份成功');
};

// 提交表单
const handleSubmit = async () => {
  try {
    submitting.value = true
    await formRef.value?.validate()
    
    // 构建提交数据
    const submitData = {
      ...formData,
      user_quote_payment: parseFloat(nationalPrice.value) || 0,
      user_quote_type: 1,
      statsu :2
    }
    
    console.log('提交数据:', submitData)
    
    // 调用后端API更新商品价格
    if (editType.value === 'price') {
      await updateBeeProductPrice(accountId.value, submitData)
    } else {
      // 省份配置更新
      const provinceData = {
        goods_id: formData.goods_id,
        provs: selectedProvinces.value,
        user_quote_type: 1,
        statsu :2
      }
      await updateBeeProductProvince(accountId.value, provinceData)
    }
    
    visible.value = false
    window.$message?.success('保存成功')
    
    // 触发父组件刷新数据
    emit('success')
  } catch (error) {
    console.error('提交失败:', error)
    window.$message?.error('保存失败')
  } finally {
    submitting.value = false
  }
}

// 取消
const handleCancel = () => {
  visible.value = false
  resetForm()
}

// 打开弹窗
const open = (id: number, product: BeeProduct, type: 'price' | 'province') => {
  accountId.value = id;
  editType.value = type;
  productName.value = product.goods_name;
  
  if (type === 'price') {
    Object.assign(formData, {
      goods_id: product.goods_id,
      status: product.status,
      prov_limit_type: product.prov_limit_type,
      user_quote_type: product.user_quote_type,
      external_code_link_type: product.external_code_link_type,
      user_quote_payment: product.user_quote_payment,
      external_code: product.external_code,
      prov_info: product.prov_info ? [...product.prov_info] : []
    });
    // 初始化选中的省份
    selectedProvinces.value = product.prov_info ? product.prov_info.map(item => item.prov) : [];
    
    // 初始化省份报价和状态数据
    provinceQuotes.value = {};
    provinceStatus.value = {};
    if (product.prov_info) {
      product.prov_info.forEach(item => {
        provinceQuotes.value[item.prov] = item.user_quote_payment?.toString() || '';
        provinceStatus.value[item.prov] = item.status === 1;
      });
    }
  } else {
    Object.assign(provinceFormData, {
      goods_id: product.goods_id,
      provs: product.prov_info ? product.prov_info.map(item => item.prov) : []
    });
  }
  
  visible.value = true;
};

defineExpose({
  open
});
</script>