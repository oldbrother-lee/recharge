import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getProductInterfaces as fetchProductInterfaces, updateProductInterfaces as updateProductInterfacesAPI, getAllInterfaces as fetchAllInterfaces } from '@/api/interface'
import type { PaginationParams } from '@/api/interface'

export interface Interface {
  id: number
  name: string
  status: number
  type: number
  created_at: string
  updated_at: string
  isp?: string
}

export const useInterfaceStore = defineStore('interface', () => {
  const productInterfaces = ref<Interface[]>([])
  const allInterfaces = ref<Interface[]>([])
  const total = ref(0)

  const getProductInterfaces = async (productId: number, params?: PaginationParams): Promise<Interface[]> => {
    try {
      const interfaces = await fetchProductInterfaces(productId, params)
      productInterfaces.value = interfaces
      console.log(productInterfaces.value,"bbbb");
      return interfaces
    } catch (error) {
      console.error('Failed to fetch product interfaces:', error)
      throw error
    }
  }

  const getAllInterfaces = async (params?: PaginationParams): Promise<Interface[]> => {
    try {
      const interfaces = await fetchAllInterfaces(params)
      allInterfaces.value = interfaces
      return interfaces
    } catch (error) {
      console.error('Failed to fetch all interfaces:', error)
      throw error
    }
  }

  const updateProductInterfaces = async (productId: number, interfaceIds: number[]): Promise<void> => {
    try {
      await updateProductInterfacesAPI(productId, interfaceIds)
    } catch (error) {
      console.error('Failed to update product interfaces:', error)
      throw error
    }
  }

  return {
    productInterfaces,
    allInterfaces,
    total,
    getProductInterfaces,
    getAllInterfaces,
    updateProductInterfaces
  }
}) 