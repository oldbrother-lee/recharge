import axios from 'axios';
import type { AxiosRequestConfig } from 'axios';
import { useMessage } from 'naive-ui';

const message = useMessage();
const instance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
});

// 请求拦截器
instance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
instance.interceptors.response.use(
  (response) => {
    const { data } = response;
    if (data.code === 200) {
      return data;
    }
    message.error(data.message || '请求失败');
    return Promise.reject(data);
  },
  (error) => {
    message.error(error.message || '请求失败');
    return Promise.reject(error);
  }
);

export const get = <T = any>(url: string, params?: any, config?: AxiosRequestConfig) => {
  return instance.get<T>(url, { params, ...config });
};

export const post = <T = any>(url: string, data?: any, config?: AxiosRequestConfig) => {
  return instance.post<T>(url, data, config);
};

export const put = <T = any>(url: string, data?: any, config?: AxiosRequestConfig) => {
  return instance.put<T>(url, data, config);
};

export const del = <T = any>(url: string, config?: AxiosRequestConfig) => {
  return instance.delete<T>(url, config);
}; 