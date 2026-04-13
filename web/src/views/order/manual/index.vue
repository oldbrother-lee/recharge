<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  NButton,
  NCard,
  NForm,
  NFormItem,
  NInput,
  NSelect,
  NSpace,
  useMessage,
  type FormInst,
  type FormRules
} from 'naive-ui';
import { useProductStore } from '@/stores/product';
import { createAgentManualOrder } from '@/api/order-manual';

defineOptions({ name: 'OrderManual' });

const message = useMessage();
const productStore = useProductStore();
const formRef = ref<FormInst | null>(null);
const loading = ref(false);

const form = ref({
  mobile: '',
  product_id: null as number | null,
  out_trade_num: '',
  isp: null as number | null,
  remark: ''
});

const rules: FormRules = {
  mobile: [{ required: true, message: '请输入充值手机号', trigger: 'blur' }],
  product_id: [{ type: 'number', required: true, message: '请选择商品', trigger: 'change' }],
  isp: [
    {
      validator(_rule, value) {
        if (value === null || value === undefined) {
          return new Error('请选择运营商');
        }
        if (![1, 2, 3].includes(Number(value))) {
          return new Error('请选择移动、电信或联通');
        }
        return true;
      },
      trigger: ['change', 'blur']
    }
  ],
  out_trade_num: [{ required: true, message: '请输入外部订单号（防重复）', trigger: 'blur' }]
};

const productOptions = computed(() =>
  (productStore.products || [])
    .filter(p => p.status === 1)
    .map(p => ({
      label: `${p.name}（¥${Number(p.price).toFixed(2)}）`,
      value: p.id
    }))
);

const ispOptions = [
  { label: '移动', value: 1 },
  { label: '电信', value: 2 },
  { label: '联通', value: 3 }
];

function fillDefaultOutTradeNum() {
  form.value.out_trade_num = `MAN${Date.now()}${Math.random().toString(36).slice(2, 8)}`;
}

function applyIspFromProduct(productId: number | null) {
  if (productId == null) {
    form.value.isp = null;
    return;
  }
  const p = productStore.products.find(x => x.id === productId);
  if (!p?.isp) {
    form.value.isp = null;
    return;
  }
  const nums = String(p.isp)
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
    .map(Number)
    .filter(n => [1, 2, 3].includes(n));
  const uniq = [...new Set(nums)];
  if (uniq.length === 1) {
    form.value.isp = uniq[0]!;
  } else {
    form.value.isp = null;
  }
}

watch(
  () => form.value.product_id,
  id => {
    applyIspFromProduct(id);
  }
);

onMounted(async () => {
  try {
    await productStore.fetchProducts();
  } catch {
    message.error('加载商品列表失败');
  }
  fillDefaultOutTradeNum();
});

async function handleSubmit() {
  await formRef.value?.validate();
  if (form.value.product_id == null) {
    message.warning('请选择商品');
    return;
  }
  if (form.value.isp === null || form.value.isp === undefined || ![1, 2, 3].includes(Number(form.value.isp))) {
    message.warning('请选择运营商');
    return;
  }
  loading.value = true;
  try {
    const res = await createAgentManualOrder({
      mobile: form.value.mobile.trim(),
      product_id: form.value.product_id,
      out_trade_num: form.value.out_trade_num.trim(),
      isp: Number(form.value.isp),
      remark: form.value.remark.trim() || undefined
    });
    const o = res.data as { order_number?: string; out_trade_num?: string };
    message.success(`下单成功，系统订单号：${o?.order_number || ''}`);
    form.value.mobile = '';
    form.value.remark = '';
    form.value.isp = null;
    fillDefaultOutTradeNum();
    if (form.value.product_id != null) {
      applyIspFromProduct(form.value.product_id);
    }
  } catch (e: any) {
    message.error(e?.message || '下单失败');
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto p-16px">
    <NCard :title="$t('route.order_manual')" :bordered="false" size="small" class="card-wrapper max-w-720px">
      <p class="mb-16px text-gray-500 text-14px">
        使用当前登录代理商账号余额/授信扣款，流程与 API 下单一致（待充值、自动进充值队列）。必须选择运营商；若商品仅支持一家运营商会自动带出。外部订单号请勿与历史订单重复。
      </p>
      <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100">
        <NFormItem label="充值手机号" path="mobile">
          <NInput v-model:value="form.mobile" placeholder="11 位手机号" maxlength="11" clearable />
        </NFormItem>
        <NFormItem label="商品" path="product_id">
          <NSelect
            v-model:value="form.product_id"
            placeholder="选择商品"
            filterable
            clearable
            :options="productOptions"
            class="w-full"
          />
        </NFormItem>
        <NFormItem label="运营商" path="isp">
          <NSelect v-model:value="form.isp" placeholder="必选：移动 / 电信 / 联通" :options="ispOptions" class="w-full" />
        </NFormItem>
        <NFormItem label="外部订单号" path="out_trade_num">
          <NInput v-model:value="form.out_trade_num" placeholder="唯一标识，防重复下单" clearable />
        </NFormItem>
        <NFormItem label="备注" path="remark">
          <NInput v-model:value="form.remark" type="textarea" placeholder="选填" :rows="2" />
        </NFormItem>
        <NFormItem>
          <NSpace>
            <NButton type="primary" :loading="loading" @click="handleSubmit">提交订单</NButton>
            <NButton @click="fillDefaultOutTradeNum">重新生成外部单号</NButton>
          </NSpace>
        </NFormItem>
      </NForm>
    </NCard>
  </div>
</template>
