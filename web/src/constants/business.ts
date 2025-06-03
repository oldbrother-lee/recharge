import { transformRecordToOption } from '@/utils/common';
import type { SelectOption } from 'naive-ui'

export const enableStatusRecord: Record<Api.Common.EnableStatus, App.I18n.I18nKey> = {
  '1': 'page.manage.common.status.enable',
  '2': 'page.manage.common.status.disable'
};

export const enableStatusOptions = transformRecordToOption(enableStatusRecord);

export const userGenderRecord: Record<Api.SystemManage.UserGender, App.I18n.I18nKey> = {
  '1': 'page.manage.user.gender.male',
  '2': 'page.manage.user.gender.female'
};

export const userGenderOptions = transformRecordToOption(userGenderRecord);

export const menuTypeRecord: Record<Api.SystemManage.MenuType, App.I18n.I18nKey> = {
  '1': 'page.manage.menu.type.directory',
  '2': 'page.manage.menu.type.menu'
};

export const menuTypeOptions = transformRecordToOption(menuTypeRecord);

export const menuIconTypeRecord: Record<Api.SystemManage.IconType, App.I18n.I18nKey> = {
  '1': 'page.manage.menu.iconType.iconify',
  '2': 'page.manage.menu.iconType.local'
};

export const menuIconTypeOptions = transformRecordToOption(menuIconTypeRecord);

// 运营商配置
export const ISP_OPTIONS: SelectOption[] = [
  { label: '移动', value: '1' },
  { label: '联通', value: '3' },
  { label: '电信', value: '2' }
]

// 运营商映射
export const ISP_MAP: Record<string, string> = {
  '1': '移动',
  '2': '电信',
  '3': '联通'
};

// 格式化运营商显示
export function formatISP(isp: string): string {
  if (!isp) return '-'
  const isps = isp.split(',')
  return isps.map(i => {
    const option = ISP_OPTIONS.find(opt => opt.value === i)
    return option ? option.label : i
  }).join('、')
}

// 商品状态配置
export const PRODUCT_STATUS_OPTIONS: SelectOption[] = [
  { label: '启用', value: 1 },
  { label: '禁用', value: 0 }
]

// 商品状态映射
export const PRODUCT_STATUS_MAP: Record<number, string> = {
  1: '启用',
  0: '禁用'
};

// 商品类型配置
export const PRODUCT_TYPE_OPTIONS = [
  { label: '话费充值', value: 1 },
  { label: '流量充值', value: 2 }
] as const;

// API 失败处理方式
export const API_FAIL_STYLE_OPTIONS = [
  { label: '自动重试', value: 1 },
  { label: '手动重试', value: 2 }
] as const;

// 展示样式
export const SHOW_STYLE_OPTIONS = [
  { label: '普通展示', value: 1 },
  { label: '特殊展示', value: 2 },
  { label: '隐藏展示', value: 3 }
] as const;
