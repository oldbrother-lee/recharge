import { request } from '@/service/request'

// 获取任务配置列表
export function getTaskConfigList(params: { page: number; page_size: number }) {
  return request({
    url: '/task-config',
    method: 'GET',
    params
  })
}

// 删除任务配置
export function deleteTaskConfig(id: number) {
  return request({
    url: `/task-config/${id}`,
    method: 'DELETE'
  })
}

// 新增任务配置
export function createTaskConfig(data: any) {
  return request({
    url: '/task-config',
    method: 'POST',
    data
  })
}

// 更新任务配置
export function updateTaskConfig(data: any) {
  return request({
    url: '/task-config',
    method: 'PUT',
    data
  })
}

// 根据ID获取任务配置
export function getTaskConfigById(id: number) {
  return request({
    url: `/task-config/${id}`,
    method: 'GET'
  })
}
