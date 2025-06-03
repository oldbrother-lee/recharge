<script setup lang="tsx">
import { ref, onMounted } from 'vue';
import { useTable } from '@/hooks/useTable';
import { useModal } from '@/hooks/useModal';
import { useForm } from '@/hooks/useForm';
import { useMessage } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import type { UserGrade, UserGradeListRequest, UserGradeUpdateRequest, UserGradeCreateRequest } from '@/typings/api/user-grade';
import { NButton, NPopconfirm, NTag, NCard, NForm, NFormItem, NSpace, NInput, NSelect } from 'naive-ui';
import { useAppStore } from '@/store/modules/app';
import { getUserGradeList, createUserGrade, updateUserGrade, deleteUserGrade } from '@/api/user-grade';

const appStore = useAppStore();
const message = useMessage();

// 获取用户等级列表
const fetchUserGrades = async () => {
  try {
    loading.value = true;
    const { page, pageSize } = pagination.value;
    
    // 过滤掉空值参数
    const searchParams = Object.fromEntries(
      Object.entries(searchForm.value).filter(([_, value]) => {
        if (value === null || value === undefined) return false;
        if (typeof value === 'string' && value.trim() === '') return false;
        return true;
      })
    );

    const params: UserGradeListRequest = {
      page: page,
      page_size: pageSize,
      ...searchParams
    };

    const res = await getUserGradeList(params);
    if (res.data) {
      data.value = res.data;
      pagination.value.itemCount = res.data.length;
    }
  } catch (error) {
    message.error('获取用户等级列表失败');
  } finally {
    loading.value = false;
  }
};

// 使用 useTable 配置
const { loading, data, pagination, handlePageChange, handlePageSizeChange, handleSearch } = useTable<UserGrade>();

// 使用 useModal 配置
const { visible, showModal, hideModal } = useModal();

// 使用 useForm 配置
const { formRef, formModel, rules, handleSubmit, resetForm } = useForm();

// 初始化表单模型
formModel.value = {
  name: '',
  description: '',
  icon: '',
  grade_type: 1,
  status: 1
};

// 初始化表单验证规则
rules.value = {
  name: {
    required: true,
    message: '请输入等级名称',
    trigger: 'blur'
  },
  grade_type: {
    required: true,
    message: '请选择等级类型',
    trigger: 'change',
    type: 'number'
  },
  status: {
    required: true,
    message: '请选择状态',
    trigger: 'change',
    type: 'number'
  }
};

// 表格列定义
const columns: DataTableColumns<UserGrade> = [
  {
    type: 'selection',
    align: 'center',
    width: 48
  },
  {
    key: 'name',
    title: '等级名称',
    align: 'center',
    minWidth: 100
  },
  {
    key: 'description',
    title: '描述',
    align: 'center',
    minWidth: 200
  },
  {
    key: 'grade_type',
    title: '等级类型',
    align: 'center',
    width: 100,
    render(row: UserGrade) {
      const typeMap: Record<number, string> = {
        1: '普通用户',
        2: '代理商',
        3: '管理员'
      };
      return typeMap[row.grade_type] || '未知';
    }
  },
  {
    key: 'status',
    title: '状态',
    align: 'center',
    width: 80,
    render(row: UserGrade) {
      const tagMap: Record<number, 'success' | 'warning'> = {
        1: 'success',
        0: 'warning'
      };
      return <NTag type={tagMap[row.status]}>{row.status === 1 ? '启用' : '禁用'}</NTag>;
    }
  },
  {
    key: 'created_at',
    title: '创建时间',
    align: 'center',
    width: 180,
    render(row: UserGrade) {
      return new Date(row.created_at).toLocaleString();
    }
  },
  {
    key: 'operate',
    title: '操作',
    align: 'center',
    width: 130,
    render(row: UserGrade) {
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

// 搜索表单
const searchForm = ref({
  name: '',
  status: null,
  grade_type: null
});

// 编辑用户等级
const handleEdit = (row: UserGrade) => {
  formModel.value = { ...row };
  showModal();
};

// 删除用户等级
const handleDelete = async (row: UserGrade) => {
  try {
    await deleteUserGrade(row.id);
    message.success('删除成功');
    fetchUserGrades();
  } catch (error) {
    message.error('删除失败');
  }
};

// 提交表单
const handleFormSubmit = async () => {
  try {
    await handleSubmit();
    if (formModel.value.id) {
      const updateData: UserGradeUpdateRequest = {
        id: formModel.value.id,
        name: formModel.value.name,
        description: formModel.value.description,
        icon: formModel.value.icon,
        grade_type: formModel.value.grade_type,
        status: formModel.value.status
      };
      await updateUserGrade(updateData);
      message.success('更新成功');
    } else {
      const createData: UserGradeCreateRequest = {
        name: formModel.value.name,
        description: formModel.value.description,
        icon: formModel.value.icon,
        grade_type: formModel.value.grade_type,
        status: formModel.value.status
      };
      await createUserGrade(createData);
      message.success('创建成功');
    }
    hideModal();
    fetchUserGrades();
  } catch (error) {
    message.error('操作失败');
  }
};

// 重置搜索表单
const handleReset = () => {
  searchForm.value = {
    name: '',
    status: null,
    grade_type: null
  };
  fetchUserGrades();
};

onMounted(() => {
  fetchUserGrades();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <!-- 搜索表单 -->
    <NCard>
      <NForm
        ref="searchFormRef"
        :model="searchForm"
        inline
        label-placement="left"
        label-width="auto"
        class="flex flex-wrap gap-16px"
      >
        <NFormItem label="等级名称" path="name">
          <NInput v-model:value="searchForm.name" placeholder="请输入等级名称" />
        </NFormItem>
        <NFormItem label="等级类型" path="grade_type">
          <NSelect
            v-model:value="searchForm.grade_type"
            :options="[
              { label: '普通用户', value: 1 },
              { label: '代理商', value: 2 },
              { label: '管理员', value: 3 }
            ]"
            placeholder="请选择等级类型"
            clearable
          />
        </NFormItem>
        <NFormItem label="状态" path="status">
          <NSelect
            v-model:value="searchForm.status"
            :options="[
              { label: '启用', value: 1 },
              { label: '禁用', value: 0 }
            ]"
            placeholder="请选择状态"
            clearable
          />
        </NFormItem>
        <NFormItem>
          <NSpace>
            <NButton type="primary" @click="handleSearch">
              搜索
            </NButton>
            <NButton @click="handleReset">重置</NButton>
          </NSpace>
        </NFormItem>
      </NForm>
    </NCard>

    <!-- 数据表格 -->
    <NCard :title="'用户等级管理'" :bordered="false" size="small" class="sm:flex-1-hidden card-wrapper">
      <template #header-extra>
        <NSpace>
          <NButton type="primary" @click="handleReset(); showModal()">
            新增等级
          </NButton>
        </NSpace>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :flex-height="!appStore.isMobile"
        :scroll-x="962"
        remote
        :row-key="(row: UserGrade) => row.id"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        class="sm:h-full"
      />
    </NCard>

    <!-- 新增/编辑弹窗 -->
    <NModal
      v-model:show="visible"
      preset="dialog"
      :title="formModel.id ? '编辑等级' : '新增等级'"
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
        <NFormItem label="等级名称" path="name">
          <NInput v-model:value="formModel.name" placeholder="请输入等级名称" />
        </NFormItem>
        <NFormItem label="描述" path="description">
          <NInput v-model:value="formModel.description" placeholder="请输入描述" />
        </NFormItem>
        <NFormItem label="等级类型" path="grade_type">
          <NSelect
            v-model:value="formModel.grade_type"
            :options="[
              { label: '普通用户', value: 1 },
              { label: '代理商', value: 2 },
              { label: '管理员', value: 3 }
            ]"
            placeholder="请选择等级类型"
          />
        </NFormItem>
        <NFormItem label="状态" path="status">
          <NSelect
            v-model:value="formModel.status"
            :options="[
              { label: '启用', value: 1 },
              { label: '禁用', value: 0 }
            ]"
            placeholder="请选择状态"
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
  </div>
</template>

<style scoped>
</style> 