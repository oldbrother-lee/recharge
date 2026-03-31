import { request } from '@/service/request';

// 获取平台账号变体配置列表（得众平台）
export function getTaskConfigList(params: { page: number; page_size: number; platform_account_id?: number }) {
  return request({
    url: '/platform/account/variants',
    method: 'GET',
    params
  });
}

// 获取闲赚侠任务配置列表
export function getXianzhuanxiaTaskConfigList(params: {
  page: number;
  page_size: number;
  platform_account_id?: number;
}) {
  return request({
    url: '/task-config',
    method: 'GET',
    params
  });
}

// 删除平台账号变体配置（得众平台）
export function deleteTaskConfig(id: number) {
  return request({
    url: `/platform/account/variants/${id}`,
    method: 'DELETE'
  });
}

// 删除闲赚侠任务配置
export function deleteXianzhuanxiaTaskConfig(id: number) {
  return request({
    url: `/task-config/${id}`,
    method: 'DELETE'
  });
}

// 新增平台账号变体配置（得众平台）
export function createTaskConfig(data: any) {
  return request({
    url: '/platform/account/variants',
    method: 'POST',
    data
  });
}

// 新增闲赚侠任务配置
export function createXianzhuanxiaTaskConfig(data: any) {
  return request({
    url: '/task-config',
    method: 'POST',
    data
  });
}

// 更新平台账号变体配置（得众平台）
export function updateTaskConfig(data: any) {
  return request({
    url: `/platform/account/variants/${data.id}`,
    method: 'PUT',
    data
  });
}

// 更新闲赚侠任务配置
export function updateXianzhuanxiaTaskConfig(data: any) {
  return request({
    url: '/task-config',
    method: 'PUT',
    data
  });
}

// 根据ID获取平台账号变体配置
export function getTaskConfigById(id: number) {
  return request({
    url: `/platform/account/variants/${id}`,
    method: 'GET'
  });
}
