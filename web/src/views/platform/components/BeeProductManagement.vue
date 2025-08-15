<template>
  <NModal
    v-model:show="visible"
    preset="dialog"
    title="商品管理"
    :style="{ width: '90%' }"
    @close="close"
  >
    <div class="product-management">
      <!-- 搜索区域 -->
      <NCard class="mb-4" :bordered="false">
        <NGrid :cols="24" :x-gap="12" :y-gap="12">
          <NGridItem :span="24" :md="6">
            <NInput
              v-model:value="searchKeyword"
              placeholder="请输入商品名称或编码"
              clearable
              @keyup.enter="handleSearch"
            />
          </NGridItem>
          <NGridItem :span="24" :md="5">
            <NSelect
              v-model:value="searchStatus"
              :options="statusOptions"
              placeholder="选择状态"
              clearable
            />
          </NGridItem>
          <NGridItem :span="24" :md="5">
            <NSelect
              v-model:value="searchType"
              :options="typeOptions"
              placeholder="选择类型"
              clearable
            />
          </NGridItem>
          <NGridItem :span="24" :md="8">
            <NSpace>
              <NButton type="primary" @click="handleSearch">搜索</NButton>
              <NButton @click="handleReset">重置</NButton>
              <NButton @click="handleRefresh">刷新</NButton>
            </NSpace>
          </NGridItem>
        </NGrid>
      </NCard>

      <!-- 统计信息 -->
       <NCard class="mb-4" :bordered="false">
         <template #header>
           <span style="font-weight: 600;">数据统计</span>
         </template>
         <NGrid :cols="3" :x-gap="12">
           <NGridItem>
             <NStatistic label="总商品数" :value="total">
               <template #prefix>
                 <NIcon color="#18a058">
                   <svg viewBox="0 0 24 24">
                     <path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                   </svg>
                 </NIcon>
               </template>
             </NStatistic>
           </NGridItem>
           <NGridItem>
             <NStatistic label="在线商品" :value="onlineCount">
               <template #prefix>
                 <NIcon color="#2080f0">
                   <svg viewBox="0 0 24 24">
                     <path fill="currentColor" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                   </svg>
                 </NIcon>
               </template>
             </NStatistic>
           </NGridItem>
           <NGridItem>
             <NStatistic label="离线商品" :value="offlineCount">
               <template #prefix>
                 <NIcon color="#d03050">
                   <svg viewBox="0 0 24 24">
                     <path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm5 11H7v-2h10v2z"/>
                   </svg>
                 </NIcon>
               </template>
             </NStatistic>
           </NGridItem>
         </NGrid>
       </NCard>

      <!-- 商品列表 -->
      <NCard :bordered="false">
        <template #header>
          <span style="font-weight: 600;">商品列表</span>
        </template>
        <NDataTable
          :columns="columns"
          :data="filteredProductList"
          :loading="loading"
          :pagination="pagination"
          :scroll-x="1300"
          size="small"
          :bordered="false"
          :single-line="false"
        />
      </NCard>
    </div>

    <!-- 价格编辑弹窗 -->
    <ProductPriceForm ref="priceFormRef" @success="fetchProductList" />
  </NModal>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, h } from 'vue';
import type { DataTableColumns } from 'naive-ui';
import { NButton, NTag, NSpace, NSelect, NCard, NGrid, NGridItem, NStatistic, NIcon, NInput, NModal, NSwitch } from 'naive-ui';
import { getBeeProductList, type BeeProduct, editBeeSupplyGoodManageStock } from '@/api/bee-platform';
import ProductPriceForm from './ProductPriceForm.vue';
import type { ComponentPublicInstance } from 'vue';

// 定义ProductPriceForm组件类型
type ProductPriceFormInstance = ComponentPublicInstance & {
  open: (accountId: number, product: BeeProduct, type: 'price' | 'province') => void;
};

interface Props {
  accountId?: number;
}

const props = defineProps<Props>();
const currentAccountId = ref<number>();

const visible = ref(false);
const loading = ref(false);
const searchKeyword = ref('');
const searchStatus = ref<number | null>(null);
const searchType = ref<number | null>(null);
const productList = ref<BeeProduct[]>([]);
const total = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);
const priceFormRef = ref<ProductPriceFormInstance>();

// 状态选项
const statusOptions = [
  { label: '上架', value: 1 },
  { label: '下架', value: 0 }
];

// 类型选项
const typeOptions = [
  { label: '话费', value: 1 },
  { label: '流量', value: 2 },
  { label: '其他', value: 3 }
];

// 计算属性：过滤后的商品列表（展开省份数据）
const filteredProductList = computed(() => {
  const filtered = productList.value.filter(product => {
    const matchKeyword = !searchKeyword.value || 
      product.goods_name?.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
      product.goods_id?.toString().includes(searchKeyword.value);
    
    const matchStatus = searchStatus.value === null || product.status === searchStatus.value;
    const matchType = searchType.value === null || product.user_quote_type === searchType.value;
    
    return matchKeyword && matchStatus && matchType;
  });
  
  // 展开有多个省份的商品为多行
  const expandedList: (BeeProduct & { _isExpanded?: boolean; _provinceName?: string; _provinceData?: any; _isSummaryRow?: boolean; _status?: number })[] = [];
  
  filtered.forEach(product => {
    const provInfo = product.user_quote_stock_prov_info;
    
    if (!provInfo || provInfo.length === 0) {
      // 没有省份信息，显示为全国
      expandedList.push({
        ...product,
        _isExpanded: false,
        _provinceName: '全国'
      });
    } else if (provInfo.length === 1) {
      // 只有一个省份，正常显示
      expandedList.push({
        ...product,
        _isExpanded: false,
        _provinceName: provInfo[0].prov,
        _provinceData: provInfo[0]
      });
    } else {
      // 多个省份，先添加汇总行，再展开为多行
      // 添加汇总行
      
      expandedList.push({
        ...product,
        _isExpanded: false,
        _provinceName: `支持${provInfo.length}个省份`,
        _isSummaryRow: true
      });
      
      // 添加各省份详细行
      provInfo.forEach((prov) => {
        expandedList.push({
          ...product,
          _isExpanded: true,
          _provinceName: prov.prov,
          _provinceData: prov,
          _isSummaryRow: false,
          _status: prov.status,
          // 省份详细行不显示商品ID和商品名称
          goods_id: product.goods_id,
          goods_name: ''
        });
      });
    }
  });
  
  console.log('过滤后的数据:', expandedList);
  return expandedList;
});

// 统计数据
const onlineCount = computed(() => {
  return productList.value.filter(item => item.status === 1).length;
});

const offlineCount = computed(() => {
  return productList.value.filter(item => item.status === 0).length;
});

// 分页配置
const pagination = computed(() => ({
  page: currentPage.value,
  pageSize: pageSize.value,
  itemCount: total.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  onChange: (page: number) => {
    currentPage.value = page;
    fetchProductList();
  },
  onUpdatePageSize: (size: number) => {
    pageSize.value = size;
    currentPage.value = 1;
    fetchProductList();
  }
}));

// 表格列定义
const columns: DataTableColumns<BeeProduct> = [
  {
    title: '商品ID',
    key: 'goods_id',
    width: 100,
    render(row: any) {
      // 只有多省份商品的详细行不显示商品ID，其他都显示
      return (!row._isExpanded || row._isSummaryRow) ? row.goods_id : '';
    }
  },
  {
    title: '渠道名称',
    key: 'vender_name',
    width: 200,
    render(row: any) {
      // 只有多省份商品的详细行不显示渠道名称，其他都显示
      return (!row._isExpanded || row._isSummaryRow) ? row.vender_name : '';
    },
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: '商品名称1',
    key: 'goods_name',
    width: 100,
    render(row: any) {
      // 只有多省份商品的详细行不显示商品名称，其他都显示
      return (!row._isExpanded || row._isSummaryRow) ? row.goods_name+'|'+row.spec_name : '';
    }
  },
  {
    title: '省份管理',
    key: 'user_quote_stock_prov_info',
    width: 100,
    render(row: any) {
      const provinceName = row._provinceName || '全国';
      const tagType = provinceName === '全国' ? 'info' : 'success';
      return h(NTag, { type: tagType, size: 'small' }, provinceName);
    }
  },
  {
    title: '最近一次成交',
    key: 'last_traded_price',
    width: 120,
    render(row: any) {
      // 优先使用省份数据中的最近成交价
      let lastPrice = null;
      if (row._provinceData && row._provinceData.last_traded_price !== undefined) {
        lastPrice = row._provinceData.last_traded_price;
      } else {
        lastPrice = row.last_traded_price;
      }
      
      if (!lastPrice || lastPrice === null || lastPrice === undefined || lastPrice === 0) {
        return '暂无数据';
      }
      const price = parseFloat(lastPrice);
      if (isNaN(price)) {
        return '-';
      }
      return `¥${price.toFixed(3)}`;
    }
  },
  {
    title: '报价区间',
    key: 'user_payment_range',
    width: 100,
  },
  {
    title: '报价',
    key: 'user_quote_stock_info',
    width: 100,
    render(row: any) {
      // 优先使用省份数据中的报价
      let payment = null;
      if (row._provinceData && row._provinceData.user_quote_payment) {
        payment = row._provinceData.user_quote_payment;
      } else if (row.user_quote_stock_info && row.user_quote_stock_info.user_quote_payment) {
        payment = row.user_quote_stock_info.user_quote_payment;
      }
      
      if (!payment || payment === null || payment === undefined) {
        return '-';
      }
      const price = parseFloat(payment);
      if (isNaN(price)) {
        return '-';
      }
      return `¥${price.toFixed(3)}`;
    }
  },

  {
    title: '供应状态',
    key: 'status',
    width: 100,
    render(row: any) {
      // 获取当前状态
      let status;
      
      // 如果是展开的省份行，使用省份数据中的状态
      if (row._isExpanded && row._provinceData) {
        status = row._provinceData.status;
      } else if (row._isExpanded && row._status !== undefined) {
        // 使用展开行的状态
        status = row._status;
      } else {
        // 非展开行，使用商品本身的状态
        status = row.status || row.supply_status;
      }
      
      const isChecked = status === 1;
      
      return h(NSwitch, {
        value: isChecked,
        checkedValue: true,
        uncheckedValue: false,
        'onUpdate:value': (value: boolean) => handleStatusChange(row, value),
        size: 'small'
      }, {
        checked: () => row._isExpanded ? '上架' : '供应中',
        unchecked: () => row._isExpanded ? '下架' : '未供应'
      });
    }
  },

  {
    title: '外部编码',
    key: 'external_code',
    width: 120,
    render(row: any) {
      // 优先使用省份数据中的外部编码
      let externalCode = null;
      if (row._provinceData && row._provinceData.external_code) {
        externalCode = row._provinceData.external_code;
      } else {
        externalCode = row.order_limit_config.external_code;
      }
      
      return externalCode || '-';
    },
    ellipsis: {
      tooltip: true
    }
  },

  {
    title: '操作',
    key: 'actions',
    width: 200,
    fixed: 'right',
    render(row: any) {
      // 只在汇总行或非展开行显示操作按钮，多省份详细行不显示
      if (!row._isSummaryRow && row._isExpanded) {
        return '';
      }
      return h(NSpace, { size: 'small' }, [
        h(NButton, {
          type: 'primary',
          size: 'small',
          onClick: () => handleEditPrice(row)
        }, '修改报价'),
        // h(NButton, {
        //   type: 'info',
        //   size: 'small',
        //   onClick: () => handleEditProvince(row)
        // }, '省份配置')
      ]);
    }
  }
];

// 获取商品列表
const fetchProductList = async () => {
  const accountId = currentAccountId.value || props.accountId;
  if (!accountId) return;
  
  loading.value = true;
  try {
    const response = await getBeeProductList(accountId, {
      keyword: searchKeyword.value,
      status: searchStatus.value,
      type: searchType.value,
      page: currentPage.value,
      page_size: pageSize.value
    });
    
    console.log('API响应:', response);
    if (response && response.data) {
      console.log("商品数据:", response.data);
      productList.value = response.data.list || [];
      total.value = response.data.total || 0;
      console.log('设置商品列表:', productList.value.length, '条记录');
    } else {
      console.error('响应数据异常:', response);
      window.$message?.error('获取商品列表失败');
    }
  } catch (error) {
    console.error('获取商品列表失败:', error);
    window.$message?.error('获取商品列表失败');
  } finally {
    loading.value = false;
  }
};

// 搜索
const handleSearch = () => {
  currentPage.value = 1;
  fetchProductList();
};

// 重置搜索
const handleReset = () => {
  searchKeyword.value = '';
  searchStatus.value = null;
  searchType.value = null;
  currentPage.value = 1;
  fetchProductList();
};

// 刷新列表
const handleRefresh = () => {
  fetchProductList();
};

// 编辑价格
const handleEditPrice = (row: any) => {
  const accountId = currentAccountId.value || props.accountId;
  
  // 无论点击哪一行（汇总/普通），都从原始列表中找到未改动的商品数据
  const originalProduct = productList.value.find(p => p.goods_id === row.goods_id);
  
  if (originalProduct) {
    // 确保传给表单的商品对象包含 prov_info
    // 如果后端返回字段是 user_quote_stock_prov_info，则做一次兼容映射
    let provInfo = (originalProduct as any).prov_info;
    if ((!provInfo || provInfo.length === 0) && (originalProduct as any).user_quote_stock_prov_info) {
      const stockProv = (originalProduct as any).user_quote_stock_prov_info as any[];
      provInfo = stockProv.map((it: any) => ({
        prov: it.prov,
        user_quote_payment: it.user_quote_discount,
        external_code: it.external_code || '',
        status: it.status
      }));
    }

    // 如果 prov_info 已存在但缺少 external_code，尝试从 user_quote_stock_prov_info 中补齐
    if (provInfo && provInfo.length > 0 && (originalProduct as any).user_quote_stock_prov_info) {
      const stockProv = (originalProduct as any).user_quote_stock_prov_info as any[];
      const externalMap = new Map<string, string>();
      stockProv.forEach((it: any) => {
        if (it && typeof it.prov !== 'undefined') {
          externalMap.set(it.prov, it.external_code || '');
        }
      });
      provInfo = provInfo.map((item: any) => ({
        ...item,
        external_code: item.external_code ?? externalMap.get(item.prov) ?? ''
      }));
    }

    const editProduct: any = {
      ...originalProduct,
      // 用规范化后的 prov_info 覆盖/补充，保证表单能识别
      ...(provInfo ? { prov_info: provInfo } : {})
    };

    priceFormRef.value?.open(accountId!, editProduct, 'price');
  } else {
    console.error('Failed to find the original product for editing.', row);
    priceFormRef.value?.open(accountId!, row, 'price');
  }
};

// 编辑省份配置
const handleEditProvince = (product: BeeProduct) => {
  const accountId = currentAccountId.value || props.accountId;
  priceFormRef.value?.open(accountId!, product, 'province');
};

// 处理供应状态变更
const handleStatusChange = async (row: any, newStatus: boolean) => {
  try {
    const accountId = currentAccountId.value || props.accountId;
    if (!accountId) {
      window.$message?.error('账户ID不能为空');
      return;
    }

    const status = newStatus ? 1 : 2; // 1开启供应，2关闭供应
    const requestData = {
      goods: [{
        goods_id: row.goods_id,
        status: status
      }]
    };

    await editBeeSupplyGoodManageStock(accountId, requestData);
    window.$message?.success('供应状态修改成功');
    
    // 更新本地数据
    if (row._isExpanded && row._provinceData) {
      // 如果是展开的省份行，更新省份数据的状态
      row._provinceData.status = status;
      row._status = status;
      
      // 同时更新原始商品数据中对应省份的状态
      const originalProduct = productList.value.find(p => p.goods_id === row.goods_id);
      if (originalProduct && originalProduct.user_quote_stock_prov_info) {
        const provInfo = originalProduct.user_quote_stock_prov_info.find(p => p.prov === row._provinceName);
        if (provInfo) {
          provInfo.status = status;
        }
      }
    } else {
      // 非展开行，更新商品本身的状态
      row.status = status;
      if (row.user_quote_stock_prov_info && row.user_quote_stock_prov_info.length > 0) {
         // 更新所有省份的状态
         row.user_quote_stock_prov_info.forEach((item: any) => {
           item.status = status;
         });
      }
    }
  } catch (error: any) {
    window.$message?.error(error.message || '供应状态修改失败');
    // 恢复原状态
    fetchProductList();
  }
};

// 打开弹窗
const open = (accountId: number) => {
  currentAccountId.value = accountId;
  visible.value = true;
  nextTick(() => {
    fetchProductList();
  });
};

// 关闭弹窗
const close = () => {
  visible.value = false;
  productList.value = [];
  searchKeyword.value = '';
  currentPage.value = 1;
};

defineExpose({
  open,
  close
});
</script>

<style scoped>
.product-management {
  padding: 0;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>