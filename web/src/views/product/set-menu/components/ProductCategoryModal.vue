<template>
  <NModal
    v-model:show="visible"
    preset="dialog"
    title="商品分类管理"
    :style="{ width: '800px'}"
  >
    <div class="flex flex-col gap-16px h-[600px]">
      <!-- 工具栏 -->
      <div class="flex justify-end">
        <NButton type="primary" @click="showAddModal">
          新增分类
        </NButton>
      </div>
      <!-- 分类列表 -->
      <div class="flex-1 overflow-hidden">
        <NDataTable
          :columns="columns"
          :data="data"
          :loading="loading"
          :pagination="pagination"
          :flex-height="true"
          :scroll-x="962"
          :max-height="500"
          remote
          :row-key="row => row.id"
          @update:page="onPageChange"
          @update:page-size="onPageSizeChange"
          class="h-full"
        />
      </div>
    </div>

    <!-- 新增/编辑分类弹窗 -->
    <NModal
      v-model:show="formVisible"
      preset="dialog"
      :title="formModel.id ? '编辑分类' : '新增分类'"
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
        <NFormItem label="分类名称" path="name">
          <NInput v-model:value="formModel.name" placeholder="请输入分类名称" />
        </NFormItem>
        <NFormItem label="排序" path="sort">
          <NInputNumber v-model:value="formModel.sort" placeholder="请输入排序" />
        </NFormItem>
        <NFormItem label="商品类型" path="type">
          <NSelect
            v-model:value="formModel.type"
            :options="productTypes.map(type => ({
              label: type.type_name,
              value: type.id
            }))"
            placeholder="请选择商品类型"
          />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="hideFormModal">取消</NButton>
          <NButton type="primary" @click="handleSubmit">确定</NButton>
        </NSpace>
      </template>
    </NModal>
  </NModal>
</template>

<script setup lang="tsx">
import { ref, onMounted } from 'vue';
import { useTable } from '@/hooks/useTable';
import { useForm } from '@/hooks/useForm';
import { useMessage } from 'naive-ui';
import { request } from '@/service/request';
import type { DataTableColumns } from 'naive-ui';
import { NButton, NPopconfirm, NTag, NForm, NFormItem, NSpace, NInput, NSelect, NInputNumber, NModal, NDataTable } from 'naive-ui';
import { useAppStore } from '@/store/modules/app';

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
const visible = ref(false);
const formVisible = ref(false);
const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable<ProductCategory>();
const { formRef, formModel, rules, handleSubmit: handleFormSubmit, resetForm } = useForm();
const productTypes = ref<ProductType[]>([]);

// 设置验证规则
rules.value = {
  name: {
    required: true,
    message: '请输入分类名称',
    trigger: ['blur', 'change']
  },
  sort: {
    required: true,
    message: '请输入排序',
    trigger: ['blur', 'change'],
    type: 'number'
  },
  type: {
    required: true,
    message: '请选择类型',
    trigger: ['blur', 'change'],
    type: 'number'
  }
};

// 表格列定义
const columns: DataTableColumns<ProductCategory> = [
  {
    key: 'id',
    title: 'ID',
    align: 'center',
    width: 80
  },
  {
    key: 'name',
    title: '分类名称',
    align: 'center',
    width: 120
  },
  {
    key: 'sort',
    title: '排序',
    align: 'center',
    width: 80
  },
  {
    key: 'type',
    title: '类型',
    align: 'center',
    width: 100,
    render(row: ProductCategory) {
      const type = productTypes.value.find(t => t.id === row.type);
      return type ? (
        <NTag type="info" size="small">{type.type_name}</NTag>
      ) : (
        <NTag type="warning" size="small">未知类型</NTag>
      );
    }
  },
  {
    key: 'created_at',
    title: '创建时间',
    align: 'center',
    width: 180,
    render(row: ProductCategory) {
      return new Date(row.created_at).toLocaleString();
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center',
    width: 120,
    render(row: ProductCategory) {
      return (
        <div class="flex-center gap-8px">
          <NButton type="primary" ghost size="small" onClick={() => handleEdit(row)}>
            编辑
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
  }
];

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
      data.value = res.data.list;
      pagination.value.itemCount = res.data.total;
      console.log('Current data:', data.value);
    }
  } catch (error) {
    console.error('获取分类列表失败:', error);
    message.error('获取分类列表失败');
  } finally {
    loading.value = false;
  }
};

// 显示新增分类弹窗
const showAddModal = () => {
  resetForm();
  formVisible.value = true;
};

// 编辑分类
const handleEdit = (row: ProductCategory) => {
  formModel.value = { ...row };
  formVisible.value = true;
};

// 删除分类
const handleDelete = async (row: ProductCategory) => {
  try {
    await request({
      url: `/product/category/${row.id}`,
      method: 'DELETE'
    });
    message.success('删除成功');
    fetchCategories();
  } catch (error) {
    message.error('删除失败');
  }
};

// 隐藏表单弹窗
const hideFormModal = () => {
  formVisible.value = false;
};

// 提交表单
const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    if (formModel.value.id) {
      await request({
        url: `/product/category/${formModel.value.id}`,
        method: 'PUT',
        data: formModel.value
      });
      message.success('更新成功');
    } else {
      await request({
        url: '/product/category',
        method: 'POST',
        data: formModel.value
      });
      message.success('创建成功');
    }
    hideFormModal();
    fetchCategories();
  } catch (error) {
    if (error instanceof Error) {
      message.error(error.message || '操作失败');
    } else {
      message.error('操作失败');
    }
  }
};

// 分页变化
const onPageChange = (page: number) => {
  pagination.value.page = page;
  fetchCategories();
};

const onPageSizeChange = (pageSize: number) => {
  pagination.value.pageSize = pageSize;
  pagination.value.page = 1;
  fetchCategories();
};

// 显示弹窗
const show = () => {
  visible.value = true;
  fetchProductTypes();
  fetchCategories();
};

// 隐藏弹窗
const hide = () => {
  visible.value = false;
};

defineExpose({
  show,
  hide
});

onMounted(() => {
  fetchProductTypes();
 
});
</script>

<style scoped>
.flex-center {
  display: flex;
  align-items: center;
  justify-content: center;
}
.gap-8px {
  gap: 8px;
}
</style> 