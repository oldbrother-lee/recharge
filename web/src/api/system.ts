import { request } from '@/service/request'

// 系统配置相关接口
export const systemConfigApi = {
  // 获取系统配置列表
  getList(params?: {
    page?: number
    pageSize?: number
    configKey?: string
    configType?: string
    isEnabled?: boolean
  }) {
    return request({
      url: '/system-config',
      method: 'GET',
      params
    })
  },

  // 根据ID获取系统配置
  getById(id: number) {
    return request({
      url: `/system-config/${id}`,
      method: 'GET'
    })
  },

  // 根据Key获取系统配置
  getByKey(key: string) {
    return request({
      url: `/system-config/key/${key}`,
      method: 'GET'
    })
  },

  // 创建系统配置
  create(data: {
    configKey: string
    configValue: string
    configDesc?: string
    configType?: string
    isEnabled?: boolean
  }) {
    return request({
      url: '/system-config',
      method: 'POST',
      data
    })
  },

  // 更新系统配置
  update(id: number, data: {
    configValue: string
    configDesc?: string
    configType?: string
    isEnabled?: boolean
  }) {
    return request({
      url: `/system-config/${id}`,
      method: 'PUT',
      data
    })
  },

  // 删除系统配置
  delete(id: number) {
    return request({
      url: `/system-config/${id}`,
      method: 'DELETE'
    })
  },

  // 批量更新配置
  batchUpdate(configs: Record<string, string>) {
    return request({
      url: '/system-config/batch',
      method: 'PUT',
      data: configs
    })
  },

  // 更新系统名称
  updateSystemName(systemName: string) {
    return request({
      url: '/system-config/system-name',
      method: 'PUT',
      data: { systemName }
    })
  },

  // 获取系统名称
  getSystemName() {
    return request({
      url: '/system-config/system-name',
      method: 'GET'
    })
  },

  // 获取系统信息
  getSystemInfo() {
    return request({
      url: '/system-config/system-info',
      method: 'GET'
    })
  }
}

// 公共接口（不需要认证）
export const publicSystemApi = {
  // 获取系统基本信息
  getBasicInfo() {
    return request({
      url: '/public/system/basic-info',
      method: 'GET'
    })
  },

  // 获取系统基本信息（包含Logo）
  getSystemBasicInfo() {
    return request({
      url: '/public/system/basic-info',
      method: 'GET'
    })
  },

  // 获取系统名称
  getSystemName() {
    return request({
      url: '/public/system/name',
      method: 'GET'
    })
  }
}

// 兼容旧版API
export const systemManageApi = {
  // 获取系统配置列表
  getList(params?: any) {
    return request({
      url: '/systemManage',
      method: 'GET',
      params
    })
  },

  // 根据ID获取系统配置
  getById(id: number) {
    return request({
      url: `/systemManage/${id}`,
      method: 'GET'
    })
  },

  // 根据Key获取系统配置
  getByKey(key: string) {
    return request({
      url: `/systemManage/key/${key}`,
      method: 'GET'
    })
  },

  // 创建系统配置
  create(data: any) {
    return request({
      url: '/systemManage',
      method: 'POST',
      data
    })
  },

  // 更新系统配置
  update(id: number, data: any) {
    return request({
      url: `/systemManage/${id}`,
      method: 'PUT',
      data
    })
  },

  // 删除系统配置
  delete(id: number) {
    return request({
      url: `/systemManage/${id}`,
      method: 'DELETE'
    })
  },

  // 批量更新配置
  batchUpdate(configs: Record<string, string>) {
    return request({
      url: '/systemManage/batch',
      method: 'PUT',
      data: configs
    })
  },

  // 更新系统名称
  updateSystemName(systemName: string) {
    return request({
      url: '/systemManage/system-name',
      method: 'PUT',
      data: { systemName }
    })
  },

  // 获取系统名称
  getSystemName() {
    return request({
      url: '/systemManage/system-name',
      method: 'GET'
    })
  },

  // 获取系统信息
  getSystemInfo() {
    return request({
      url: '/systemManage/system-info',
      method: 'GET'
    })
  }
}

export default {
  systemConfigApi,
  publicSystemApi,
  systemManageApi
}