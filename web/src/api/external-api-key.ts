import { request } from '@/service/request';

// API密钥相关类型定义
export interface ExternalAPIKey {
  id: number;
  app_id: string;
  app_key: string;
  app_secret: string;
  app_name: string;
  description: string;
  status: number;
  ip_whitelist: string;
  notify_url: string;
  rate_limit: number;
  expire_time?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAPIKeyRequest {
  app_name?: string;
  description?: string;
}

export interface UpdateStatusRequest {
  status: number;
}

// 创建API密钥
export const createAPIKey = (data: CreateAPIKeyRequest) => {
  return request<ExternalAPIKey>({
    url: '/external-api-keys',
    method: 'POST',
    data
  });
};

// 获取我的API密钥
export const getMyAPIKeys = () => {
  return request<ExternalAPIKey | null>({
    url: '/external-api-keys/my',
    method: 'GET'
  });
};

// 重新生成API密钥
export const regenerateAPIKey = (id: number) => {
  return request<ExternalAPIKey>({
    url: `/external-api-keys/${id}/regenerate`,
    method: 'POST'
  });
};

// 更新API密钥状态
export const updateAPIKeyStatus = (id: number, data: UpdateStatusRequest) => {
  return request<ExternalAPIKey>({
    url: `/external-api-keys/${id}/status`,
    method: 'PUT',
    data
  });
};