import { defineStore } from 'pinia'
import { ref } from 'vue'
import { request } from '@/service/request'

export interface Product {
  id: number
  name: string
  description: string
  type: number
  category_id: number
  isp: string
  status: number
  price: number
  max_price: number
  sort: number
  api_enabled: boolean
  is_delay: number
  is_decode: boolean
  decode_api: string
  decode_api_package: string
  show_style: number
  allow_province: string
  allow_city: string
  forbid_province: string
  forbid_city: string
  created_at: string
  updated_at: string
  category?: {
    id: number
    name: string
  }
}

export const useProductStore = defineStore('product', () => {
  const products = ref<Product[]>([])

  const fetchProducts = async () => {
    try {
      const res = await request({
        url: '/product/list',
        method: 'GET'
      })
      if (res.data) {
        products.value = res.data.records
      }
    } catch (error) {
      console.error('Failed to fetch products:', error)
      throw error
    }
  }

  return {
    products,
    fetchProducts
  }
}) 