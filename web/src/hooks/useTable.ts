import { ref } from 'vue';

export function useTable<T>() {
  const loading = ref(false);
  const data = ref<T[]>([]);
  const pagination = ref({
    page: 1,
    pageSize: 10,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 30, 40],
    onChange: (page: number) => {
      pagination.value.page = page;
    },
    onUpdatePageSize: (pageSize: number) => {
      pagination.value.pageSize = pageSize;
      pagination.value.page = 1;
    }
  });

  const handlePageChange = (page: number) => {
    pagination.value.page = page;
  };

  const handlePageSizeChange = (pageSize: number) => {
    pagination.value.pageSize = pageSize;
    pagination.value.page = 1;
  };

  const handleSearch = (searchFn: () => Promise<void>) => {
    pagination.value.page = 1;
    searchFn();
  };

  return {
    loading,
    data,
    pagination,
    handlePageChange,
    handlePageSizeChange,
    handleSearch
  };
} 