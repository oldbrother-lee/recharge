import { request } from '@/service/request'

export interface Interface {
  id: number
  name: string
  status: number
  type: number
  created_at: string
  updated_at: string
}

export interface PaginationParams {
  page: number
  pageSize: number
}

export interface ProductInterfaceResponse {
  records: Interface[]
  total: number
  current: number
  size: number
}

export const getProductInterfaces = async (productId: number, params?: PaginationParams): Promise<Interface[]> => {
  try {
    const response = await request<App.Service.Response<ProductInterfaceResponse>>({
      url: `/product-api-relations?product_id=${productId}`,
      method: 'GET',
      params: {
        page: params?.page || 1,
        page_size: params?.pageSize || 10
      }
    })
    console.log(response.data,"response.data");
    return response.data.list || []
  } catch (error) {
    console.error('Failed to fetch product interfaces:', error)
    return []
  }
}

export const getAllInterfaces = async (params?: PaginationParams): Promise<Interface[]> => {
  try {
    const response = await request<App.Service.Response<ProductInterfaceResponse>>({
      url: '/platform/api',
      method: 'GET',
      params: {
        page: params?.page || 1,
        page_size: params?.pageSize || 10
      }
    })
    return response.data.list || []
  } catch (error) {
    console.error('Failed to fetch all interfaces:', error)
    return []
  }
}

export const updateProductInterfaces = async (productId: number, interfaceIds: number[]): Promise<void> => {
  try {
    await request<App.Service.Response<void>>({
      url: '/product-api-relations',
      method: 'POST',
      data: {
        product_id: productId,
        interface_ids: interfaceIds
      }
    })
  } catch (error) {
    console.error('Failed to update product interfaces:', error)
    throw error
  }
} 