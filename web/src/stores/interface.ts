import { ref } from 'vue';
import { defineStore } from 'pinia';
import {
  getAllInterfaces as fetchAllInterfaces,
  getProductInterfaces as fetchProductInterfaces,
  updateProductInterfaces as updateProductInterfacesAPI
} from '@/api/interface';

export interface Interface {
  id: number;
  name: string;
  // ... other interface fields
}

export const useInterfaceStore = defineStore('interface', () => {
  const productInterfaces = ref<Interface[]>([]);
  const allInterfaces = ref<Interface[]>([]);

  const getProductInterfaces = async (productId: number): Promise<Interface[]> => {
    try {
      const response = await fetchProductInterfaces(productId);
      productInterfaces.value = response.data;
      console.log(productInterfaces.value, 'vvvvv');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch product interfaces:', error);
      throw error;
    }
  };

  const getAllInterfaces = async (): Promise<Interface[]> => {
    try {
      const response = await fetchAllInterfaces();
      allInterfaces.value = response.data;
      return response.data;
    } catch (error) {
      console.error('Failed to fetch all interfaces:', error);
      throw error;
    }
  };

  const updateProductInterfaces = async (productId: number, interfaceIds: number[]): Promise<void> => {
    try {
      await updateProductInterfacesAPI(productId, interfaceIds);
    } catch (error) {
      console.error('Failed to update product interfaces:', error);
      throw error;
    }
  };

  return {
    productInterfaces,
    allInterfaces,
    getProductInterfaces,
    getAllInterfaces,
    updateProductInterfaces
  };
});
