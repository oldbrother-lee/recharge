<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <!-- 搜索表单 -->
    <NCard :bordered="false" size="small" class="mb-16px">
      <NForm
        ref="searchFormRef"
        :model="searchForm"
        label-placement="left"
        :label-width="80"
      >
        <NCollapse :default-expanded-names="[]">
          <NCollapseItem title="搜索条件" name="user-search">
            <NGrid responsive="screen" item-responsive :x-gap="24">
              <NFormItemGi span="24 s:12 m:6" label="商品名称" path="name">
                <NInput v-model:value="searchForm.name" placeholder="请输入商品名称" />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:6" label="商品类型" path="type">
                <NSelect
                  v-model:value="searchForm.type"
                  :options="productTypes.map(type => ({ label: type.type_name, value: type.id }))"
                  placeholder="请选择商品类型"
                  clearable
                />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:6" label="运营商" path="isp">
                <NSelect
                  v-model:value="searchForm.isp"
                  :options="ISP_OPTIONS"
                  placeholder="请选择运营商"
                  clearable
                  multiple
                  :max-tag-count="2"
                  :consistent-menu-width="false"
                />
              </NFormItemGi>
              <NFormItemGi span="24 s:12 m:6" label="状态" path="status">
                <NSelect
                  v-model:value="searchForm.status"
                  :options="PRODUCT_STATUS_OPTIONS"
                  placeholder="请选择状态"
                  clearable
                />
              </NFormItemGi>
              <NFormItemGi span="24" class="pr-24px">
                <NSpace class="w-full" justify="end">
                  <NButton @click="handleReset">重置</NButton>
                  <NButton type="primary" ghost @click="handleSearch(fetchProducts)">搜索</NButton>
                </NSpace>
              </NFormItemGi>
            </NGrid>
          </NCollapseItem>
        </NCollapse>
      </NForm>
    </NCard>

    <!-- 数据表格 -->
    <NCard :title="'商品管理'" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
      <template #header-extra>
        <NSpace>
          <NButton v-if="hasRole('SUPER_ADMIN')" type="primary" @click="showCategoryModal">
            分类管理
          </NButton>
          <NButton v-if="hasRole('SUPER_ADMIN')" type="primary" @click="handleReset(); showModal()">
            新增商品
          </NButton>
        </NSpace>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :flex-height="true"
        :scroll-x="1200"
        remote
        :row-key="row => row.id"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
        class="h-full"
        size="small"
      />
    </NCard>

    <!-- 新增/编辑弹窗 -->
    <NModal
      v-model:show="visible"
      preset="dialog"
      :title="formModel.id ? '编辑商品' : '新增商品'"
      :style="{ width: '800px' }"
    >
      <NForm
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        :label-width="80"
        require-mark-placement="right-hanging"
      >
        <NGrid :cols="24" :x-gap="24">
          <NFormItemGi :span="12" label="商品名称" path="name">
            <NInput v-model:value="formModel.name" placeholder="请输入商品名称" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="商品类型" path="type">
            <NSelect
              v-model:value="formModel.type"
              :options="productTypes.map(type => ({ label: type.type_name, value: type.id }))"
              placeholder="请选择商品类型"
            />
          </NFormItemGi>
          <NFormItemGi :span="12" label="商品分类" path="category_id">
            <NSelect
              v-model:value="formModel.category_id"
              :options="productCategories.map(category => ({ label: category.name, value: category.id }))"
              placeholder="请选择商品分类"
            />
          </NFormItemGi>
          <NFormItemGi :span="12" label="显示端" path="show_style">
            <NSelect
              v-model:value="formModel.show_style"
              :options="SHOW_STYLE_OPTIONS"
              placeholder="请选择显示端"
            />
          </NFormItemGi>
          <NFormItemGi :span="24" label="商品描述" path="description">
            <NInput v-model:value="formModel.description" type="textarea" placeholder="请输入商品描述" />
          </NFormItemGi>
          <NFormItemGi :span="24" label="运营商" path="isp">
            <NCheckboxGroup v-model:value="formModel.isp">
              <NCheckbox :value="1">移动</NCheckbox>
              <NCheckbox :value="2">电信</NCheckbox>
              <NCheckbox :value="3">联通</NCheckbox>
            </NCheckboxGroup>
          </NFormItemGi>
          <NFormItemGi :span="12" label="基础价格" path="price">
            <NInputNumber v-model:value="formModel.price" :precision="2" :step="0.1" :min="0" :show-button="false"/>
          </NFormItemGi>
          <NFormItemGi :span="12" label="封顶价格" path="max_price">
            <NInputNumber v-model:value="formModel.max_price" :precision="2" :step="0.1" :min="0" :show-button="false"/>
          </NFormItemGi>
          <NFormItemGi :span="12" label="允许下单省份">
            <NInput v-model:value="formModel.allow_province" type="textarea" placeholder="请输入允许下单省份" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="允许下单城市">
            <NInput v-model:value="formModel.allow_city" type="textarea" placeholder="请输入允许下单城市" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="禁止下单省份">
            <NInput v-model:value="formModel.forbid_province" type="textarea" placeholder="请输入禁止下单省份" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="禁止下单城市">
            <NInput v-model:value="formModel.forbid_city" type="textarea" placeholder="请输入禁止下单城市" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="状态" path="status">
            <NRadioGroup v-model:value="formModel.status">
              <NRadio :value="1">上架</NRadio>
              <NRadio :value="0">下架</NRadio>
            </NRadioGroup>
          </NFormItemGi>
          <NFormItemGi :span="12" label="排序">
            <NInputNumber v-model:value="formModel.sort" :min="0" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="接口充值" path="api_enabled">
            <NSwitch v-model:value="formModel.api_enabled" />
          </NFormItemGi>
          <NFormItemGi :span="12" label="延迟提交" path="is_delay">
            <NInputNumber v-model:value="formModel.is_delay" :min="0" :show-button="false"/>  
          </NFormItemGi>
          <NFormItemGi :span="8" label="是否接码" path="is_decode">
            <NSwitch v-model:value="formModel.is_decode" />
          </NFormItemGi>
          <NFormItemGi :span="8" label="接码api" path="decode_api">
            <NInput v-model:value="formModel.decode_api" placeholder="请输入接码api" />
          </NFormItemGi> 
          <NFormItemGi :span="8" label="接码api套餐" path="decode_api_package">
            <NInput v-model:value="formModel.decode_api_package" placeholder="请输入接码api套餐" />
          </NFormItemGi> 
        </NGrid>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="hideModal">取消</NButton>
          <NButton type="primary" @click="handleFormSubmit">确定</NButton>
        </NSpace>
      </template>
    </NModal>

    <ProductCategoryModal ref="categoryModalRef" />

    <!-- 接口选择对话框 -->
    <NModal
      v-model:show="showInterfaceDialog"
      preset="dialog"
      title="选择接口"
      :style="{ width: '1000px' }"
    >
    <div class="flex flex-col gap-16px">
        <!-- 工具栏 -->
        <div class="flex justify-end">
          <NButton type="primary" @click="handleOpenCreateRelation">
            创建关联
          </NButton>
        </div>
      </div>
    
      <NForm
        v-if="selectedProduct"
        :model="interfaceForm"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <NFormItem label="商品名称">
          <NInput :value="selectedProduct.name" disabled />
        </NFormItem>
        <NFormItem label="已绑定接口">
          <NDataTable
            :columns="interfaceColumns"
            :data="productInterfaces"
            :loading="interfaceLoading"
            :pagination="interfacePagination"
            :row-key="row => row.id"
            @update:page="onInterfacePageChange"
            @update:page-size="onInterfacePageSizeChange"
          />
        </NFormItem>
      </NForm>
    </NModal>

    <!-- 创建接口关联弹窗 -->
    <NModal
      v-model:show="showCreateRelationDialog"
      preset="dialog"
      :title="isEdit ? '编辑接口关联' : '创建接口关联'"
      :style="{ width: '600px' }"
    >
      <NForm
        ref="relationFormRef"
        :model="relationForm"
        :rules="relationRules"
        label-placement="left"
        label-width="auto"
        require-mark-placement="right-hanging"
      >
        <NFormItem label="接口" path="api_id">
          <NSelect
            v-model:value="relationForm.api_id"
            :options="interfaceOptions"
            placeholder="请选择接口"
          />
        </NFormItem>
        <NFormItem label="选择套餐" path="param_id">
          <NSelect
            v-model:value="relationForm.param_id"
            :options="packageOptions"
            placeholder="请选择套餐"
          />
        </NFormItem>
        <NFormItem label="排序" path="sort">
          <NInputNumber v-model:value="relationForm.sort" :min="0" />
        </NFormItem>
        <NFormItem label="状态" path="status">
          <NRadioGroup v-model:value="relationForm.status">
            <NRadio :value="1">启用</NRadio>
            <NRadio :value="0">禁用</NRadio>
          </NRadioGroup>
        </NFormItem>
        <NFormItem label="重试次数" path="retry_num">
          <NInputNumber v-model:value="relationForm.retry_num" :min="0" />
        </NFormItem>
        <NFormItem label="运营商" path="isp">
          <NCheckboxGroup v-model:value="relationForm.isp">
            <NCheckbox :value="1">移动</NCheckbox>
            <NCheckbox :value="2">电信</NCheckbox>
            <NCheckbox :value="3">联通</NCheckbox>
          </NCheckboxGroup>
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="showCreateRelationDialog = false">取消</NButton>
          <NButton type="primary" @click="handleCreateRelation">确定</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<script setup lang="tsx">
import { ref, onMounted, watch } from 'vue';
import { useTable } from '@/hooks/useTable';
import { useModal } from '@/hooks/useModal';
import { useForm } from '@/hooks/useForm';
import { useMessage } from 'naive-ui';
import { request } from '@/service/request';
import type { DataTableColumns } from 'naive-ui';
import type { Product } from '@/typings/api';
import { NButton, NPopconfirm, NTag, NCard, NForm, NFormItem, NSpace, NInput, NSelect, NInputNumber, NSwitch, NRadioGroup, NRadio, NCheckboxGroup, NCollapse, NCollapseItem } from 'naive-ui';
import type { FormRules } from 'naive-ui';
import { useAppStore } from '@/store/modules/app';
import { ISP_OPTIONS, PRODUCT_STATUS_OPTIONS, formatISP } from '@/constants/business';
import ProductCategoryModal from './components/ProductCategoryModal.vue';
import { useProductStore } from '@/store/modules/product';
import { useInterfaceStore, type Interface } from '@/store/modules/interface';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/store/modules/auth';

const authStore = useAuthStore();

const hasRole = (role: string) => {
  return authStore.userInfo.roles.includes(role);
};

interface ProductType {
  id: number;
  type_name: string;
  sort: number;
  created_at: string;
}

interface ProductCategory {
  id: number;
  name: string;
  sort: number;
  type: number;
  created_at: string;
}

const appStore = useAppStore();
const message = useMessage();
const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable<Product>();
const { visible, showModal, hideModal } = useModal();
const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();

const categoryModalRef = ref();
const productTypes = ref<ProductType[]>([]);
const productCategories = ref<ProductCategory[]>([]);
const SHOW_STYLE_OPTIONS = ref([
  {
    label: '全部显示',
    value: 1
  },
  {
    label: '客户端',
    value: 2
  },
  {
    label: '代理端',
    value: 3
  }
])
const columns: DataTableColumns<Product> = [
  {
    type: 'selection',
    align: 'center',
    width: 48
  },
  {
    key: 'id',
    title: '商品id',
    align: 'center',
    width: 60
  },
  {
    key: 'name',
    title: '商品名称',
    align: 'center',
    minWidth: 120
  },
  {
    key: 'type',
    title: '商品类型',
    align: 'center',
    width: 150,
    render(row) {
      const type = productTypes.value.find(t => t.id === row.type);
      return type ? type.type_name : '-';
    }
  },
  {
    key: 'category',
    title: '商品分类',
    align: 'center',
    width: 150,
    render(row) {
      return row.category?.name;
    }
  },
  {
    key: 'isp',
    title: '运营商',
    align: 'center',
    width: 150,
    render(row) {
      return formatISP(row.isp);
    }
  },
  {
    key: 'price',
    title: '价格',
    align: 'center',
    width: 50
  },
  {
    key: 'status',
    title: '状态',
    align: 'center',
    width: 150,
    render(row) {
      const tagMap: Record<number, 'success' | 'warning'> = {
        1: 'success',
        0: 'warning'
      };
      return <NTag type={tagMap[row.status]}>{row.status === 1 ? '启用' : '禁用'}</NTag>;
    }
  },
  ...(hasRole('SUPER_ADMIN') ? [{
    key: 'operate',
    title: '操作',
    align: 'center' as const,
    width: 240,
    render(row: Product) {
      return (
        <div class="flex-center gap-8px">
          <NButton type="primary" ghost size="small" onClick={() => handleEdit(row)}>
            编辑
          </NButton>
          <NButton type="primary" ghost size="small" onClick={() => handleSelectInterface(row)}>
            接口选择
          </NButton>
          <NPopconfirm onPositiveClick={() => handleDelete(row)}>
            {{
              default: () => '确认删除？',
              trigger: () => (
                <NButton type="error" ghost size="small">
                  删除
                </NButton>
              )
            }}
          </NPopconfirm>
        </div>
      );
    }
  }] : [])
];

// 搜索表单
const searchForm = ref({
  name: '',
  type: null,
  category_id: null,
  isp: null as string | null,
  status: null
});

// 获取商品类型列表
const fetchProductTypes = async () => {
  try {
    const res = await request({
      url: '/product/types',
      method: 'GET'
    });
    if (res.data) {
      productTypes.value = res.data;
    }
  } catch (error) {
    console.error('获取商品类型失败:', error);
    message.error('获取商品类型失败');
  }
};
// 获取分类列表
const fetchCategories = async () => {
  try {
    loading.value = true;
    const { page, pageSize } = pagination.value;
    
    const res = await request({
      url: '/product/categories',
      method: 'GET',
      params: {
        page,
        page_size: pageSize
      }
    });
    
    if (res.data) {
      productCategories.value = res.data.list;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    console.error('获取分类列表失败:', error);
    message.error('获取分类列表失败');
  } finally {
    loading.value = false;
  }
};
// 获取商品列表
const fetchProducts = async () => {
  try {
    loading.value = true;
    const { page, pageSize } = pagination.value;
    
    // 过滤掉空值参数
    const searchParams = Object.fromEntries(
      Object.entries(searchForm.value).filter(([_, value]) => {
        if (value === null || value === undefined) return false;
        if (Array.isArray(value) && value.length === 0) return false;
        if (typeof value === 'string' && value.trim() === '') return false;
        return true;
      })
    );

    // 处理运营商参数
    if (Array.isArray(searchParams.isp)) {
      searchParams.isp = searchParams.isp.join(',');
    }

    const params = {
      page,
      page_size: pageSize,
      ...searchParams
    };

    const res = await request({
      url: '/product/list',
      method: 'GET',
      params
    });
    if (res.data) {
      data.value = res.data.records;
      pagination.value.itemCount = res.data.total;
    }
  } catch (error) {
    message.error('获取商品列表失败');
  } finally {
    loading.value = false;
  }
};

// 编辑商品
const handleEdit = (row: Product) => {
  formModel.value = { 
    ...row,
    isp: row.isp ? row.isp.split(',').map(Number) : [] // 将字符串转换为数字数组
  };
  showModal();
};

// 删除商品
const handleDelete = async (row: Product) => {
  try {
    await request({
      url: `/product/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchProducts();
  } catch (error) {
    message.error('删除失败');
  }
};

// 提交表单
const handleFormSubmit = async () => {
  try {
    await handleSubmit();
    
    // 处理 isp 字段，将数组转换为逗号分隔的字符串
    const submitData = {
      ...formModel.value,
      isp: Array.isArray(formModel.value.isp) ? formModel.value.isp.join(',') : formModel.value.isp
    };

    console.log('提交数据', submitData);

    if (formModel.value.id) {
      await request({
        url: `/product/${formModel.value.id}`,
        method: 'PUT',
        data: submitData
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/product',
        method: 'POST',
        data: submitData
      });
      message.success('创建成功');
    }
    hideModal();
    fetchProducts();
  } catch (error) {
    message.error('操作失败');
  }
};

// 重置搜索表单
const handleReset = () => {
  searchForm.value = {
    name: '',
    type: null,
    category_id: null,
    isp: null,
    status: null
  };
  fetchProducts();
};

// 添加这些处理函数
const onPageChange = (page: number) => {
  pagination.value.page = page;
  fetchProducts();
};

const onPageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
  fetchProducts();
};

// 显示分类管理弹窗
const showCategoryModal = () => {
  categoryModalRef.value?.show();
};

const productStore = useProductStore();
const interfaceStore = useInterfaceStore();

const showInterfaceDialog = ref(false);
const selectedProduct = ref<Product | null>(null);
const selectedInterfaces = ref<number[]>([]);
const interfaceOptions = ref<{ label: string; value: number }[]>([]);
const packageOptions = ref<{ label: string; value: number }[]>([]);
const interfaceForm = ref({
  productId: 0,
  interfaceIds: [] as number[]
});

const router = useRouter();

const showCreateRelationDialog = ref(false);
const relationFormRef = ref();
const relationForm = ref({
  api_id: null as number | null,
  param_id: null as number | null,
  sort: 0,
  status: 1,
  retry_num: 0,
  isp: [] as number[]
});

const relationRules: FormRules = {
  api_id: {
    required: true,
    type: 'number',
    message: '请选择接口',
    trigger: ['blur', 'change']
  },
  param_id: {
    required: true,
    type: 'number',
    message: '请选择套餐',
    trigger: ['blur', 'change']
  },
  sort: {
    required: true,
    type: 'number',
    message: '请输入排序',
    trigger: ['blur', 'change']
  },
  status: {
    required: true,
    type: 'number',
    message: '请选择状态',
    trigger: ['blur', 'change']
  },
  retry_num: {
    required: true,
    type: 'number',
    message: '请输入重试次数',
    trigger: ['blur', 'change']
  },
  isp: {
    required: true,
    type: 'array',
    message: '请选择运营商',
    trigger: ['blur', 'change']
  }
};

const handleSelectInterface = (row: Product) => {
  showInterfaceDialog.value = true;
  selectedProduct.value = row;
  loadProductInterfaces(row.id);
  loadInterfaceOptions();
};

const interfaceLoading = ref(false);
const productInterfaces = ref<Interface[]>([]);
const interfacePagination = ref({
  page: 1,
  pageSize: 10,
  itemCount: 0
});

const interfaceColumns: DataTableColumns<Interface> = [
  {
    key: 'product_name',
    title: '渠道',
    align: 'center',
    minWidth: 150
  },
  {
    key: 'api_name',
    title: '套餐名称',
    align: 'center',
    minWidth: 150
  },
  {
    key: 'retry_num',
    title: '提交次数',
    align: 'center',
    width: 100
  },
  {
    key: 'isp',
    title: '运营商',
    align: 'center',
    width: 100,
    render(row) {
      return row.isp ? formatISP(row.isp) : '-';
    }
  },
  {
    key: 'type',
    title: '地区限制',
    align: 'center',
    width: 100
  },
  {
    key: 'status',
    title: '接口状态',
    align: 'center',
    width: 100,
    render(row) {
      const tagMap: Record<number, 'success' | 'warning'> = {
        1: 'success',
        0: 'warning'
      };
      return <NTag type={tagMap[row.status]}>{row.status === 1 ? '启用' : '禁用'}</NTag>;
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center' as const,
    width: 150,
    render(row: Interface) {
      return (
        <div class="flex-center gap-8px">
          <NButton type="primary" ghost size="small" onClick={() => handleEditRelation(row)}>
            编辑
          </NButton>
          <NPopconfirm onPositiveClick={() => handleDeleteRelation(row)}>
            {{
              default: () => '确认删除？',
              trigger: () => (
                <NButton type="error" ghost size="small">
                  删除
                </NButton>
              )
            }}
          </NPopconfirm>
        </div>
      );
    }
  }
];

const loadProductInterfaces = async (productId: number) => {
  try {
    interfaceLoading.value = true;
    const { page, pageSize } = interfacePagination.value;
    
    const interfaces = await interfaceStore.getProductInterfaces(productId, {
      page,
      pageSize
    });
    productInterfaces.value = interfaces;
    interfacePagination.value.itemCount = interfaceStore.total;
  } catch (error) {
    console.error('Failed to load product interfaces:', error);
    message.error('加载接口失败');
  } finally {
    interfaceLoading.value = false;
  }
};

const loadInterfaceOptions = async () => {
  try {
    const { page, pageSize } = interfacePagination.value;
    const interfaces = await interfaceStore.getAllInterfaces({
      page,
      pageSize
    });
    interfaceOptions.value = interfaces.map(i => ({
      label: `${i.name} (${i.type === 1 ? '充值' : '查询'})`,
      value: i.id
    }));
  } catch (error) {
    console.error('Failed to load interface options:', error);
    message.error('加载接口选项失败');
  }
};

const loadPackageOptions = async (apiId: number) => {
  try {
    const res = await request({
      url: `/platform/api/params?api_id=${apiId}`,
      method: 'GET'
    });
    if (res.data) {
      packageOptions.value = res.data.list.map((pkg: any) => ({
        label: `${pkg.name} (${pkg.price}元)`,
        value: pkg.id
      }));
    }
  } catch (error) {
    console.error('Failed to load package options:', error);
    message.error('加载套餐选项失败');
  }
};

watch(() => relationForm.value.api_id, async (newVal) => {
  if (newVal) {
    await loadPackageOptions(newVal);
  } else {
    packageOptions.value = [];
    relationForm.value.param_id = null;
  }
});

const handleSaveInterfaces = async () => {
  if (!selectedProduct.value) return;
  
  try {
    await interfaceStore.updateProductInterfaces(selectedProduct.value.id, selectedInterfaces.value);
    message.success('保存成功');
    showInterfaceDialog.value = false;
  } catch (error) {
    console.error('Failed to save product interfaces:', error);
    message.error('保存失败');
  }
};

const onInterfacePageChange = (page: number) => {
  interfacePagination.value.page = page;
  loadProductInterfaces(selectedProduct.value?.id || 0);
};

const onInterfacePageSizeChange = (pageSize: number) => {
  interfacePagination.value.pageSize = pageSize;
  interfacePagination.value.page = 1;
  loadProductInterfaces(selectedProduct.value?.id || 0);
};

const isEdit = ref(false);
const currentRelationId = ref<number | null>(null);

const handleEditRelation = (row: any) => {
  isEdit.value = true;
  currentRelationId.value = row.id;
  showCreateRelationDialog.value = true;
  relationForm.value = {
    api_id: row.api_id,
    param_id: row.param_id,
    sort: row.sort,
    status: row.status,
    retry_num: row.retry_num,
    isp: row.isp.split(',').map(Number)
  };
};

const handleCreateRelation = async () => {
  try {
    await relationFormRef.value?.validate();
    
    if (!selectedProduct.value) {
      message.warning('请先选择商品');
      return;
    }
    
    const data = {
      product_id: selectedProduct.value.id,
      ...relationForm.value,
      isp: relationForm.value.isp.join(',')
    };
    
    if (isEdit.value && currentRelationId.value) {
      await request({
        url: `/product-api-relations/${currentRelationId.value}`,
        method: 'PUT',
        data: {
          id: currentRelationId.value,
          ...data
        }
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/product-api-relations',
        method: 'POST',
        data
      });
      message.success('创建成功');
    }
    
    showCreateRelationDialog.value = false;
    loadProductInterfaces(selectedProduct.value.id);
    
    // 重置表单和状态
    relationForm.value = {
      api_id: null,
      param_id: null,
      sort: 0,
      status: 1,
      retry_num: 0,
      isp: []
    };
    isEdit.value = false;
    currentRelationId.value = null;
  } catch (error) {
    console.error('Failed to save relation:', error);
    message.error(isEdit.value ? '更新失败' : '创建失败');
  }
};

const handleOpenCreateRelation = () => {
  if (!selectedProduct.value) {
    message.warning('请先选择商品');
    return;
  }
  
  isEdit.value = false;
  currentRelationId.value = null;
  showCreateRelationDialog.value = true;
  // 重置表单
  relationForm.value = {
    api_id: null,
    param_id: null,
    sort: 0,
    status: 1,
    retry_num: 0,
    isp: []
  };
};

const handleDeleteRelation = async (row: Interface) => {
  try {
    await request({
      url: `/product-api-relations/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    loadProductInterfaces(selectedProduct.value?.id || 0);
  } catch (error) {
    console.error('Failed to delete relation:', error);
    message.error('删除失败');
  }
};

onMounted(() => {
  fetchProductTypes();
  fetchProducts();
  fetchCategories();
  loadInterfaceOptions();
});
</script>

<style scoped>
.min-h-500px {
  min-height: 500px;
}
.flex-col-stretch {
  display: flex;
  flex-direction: column;
}
.gap-16px {
  gap: 16px;
}
.lt-sm\:overflow-auto {
  @media (max-width: 640px) {
    overflow: auto;
  }
}
.overflow-hidden {
  overflow: hidden;
}
.flex-wrap {
  flex-wrap: wrap;
}
.sm\:flex-1-hidden {
  @media (min-width: 640px) {
    flex: 1;
    overflow: hidden;
  }
}
.card-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.h-full {
  height: 100%;
}
.flex-center {
  display: flex;
  align-items: center;
  justify-content: center;
}
.gap-8px {
  gap: 8px;
}
@media (max-width: 640px) {
  .n-data-table .n-data-table-td,
  .n-data-table .n-data-table-th {
    white-space: nowrap !important;
    padding-top: 4px !important;
    padding-bottom: 4px !important;
    font-size: 13px !important;
  }
  .n-data-table .n-data-table-td {
    min-height: 28px !important;
  }
}
</style> 