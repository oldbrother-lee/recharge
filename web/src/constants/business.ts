import type { SelectOption } from 'naive-ui';
import { transformRecordToOption } from '@/utils/common';

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
];

// 运营商映射
export const ISP_MAP: Record<string, string> = {
  '1': '移动',
  '2': '电信',
  '3': '联通'
};

// 格式化运营商显示
export function formatISP(isp: string): string {
  if (!isp) return '-';
  const isps = isp.split(',');
  return isps
    .map(i => {
      const option = ISP_OPTIONS.find(opt => opt.value === i);
      return option ? option.label : i;
    })
    .join('、');
}

// 商品状态配置
export const PRODUCT_STATUS_OPTIONS: SelectOption[] = [
  { label: '启用', value: 1 },
  { label: '禁用', value: 0 }
];

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

// 订单状态（与后端 OrderStatus 一致）
export const ORDER_STATUS_OPTIONS: SelectOption[] = [
  { label: '待支付', value: 1 },
  { label: '待充值', value: 2 },
  { label: '充值中', value: 3 },
  { label: '充值成功', value: 4 },
  { label: '充值失败', value: 5 },
  { label: '已退款', value: 6 },
  { label: '已取消', value: 7 },
  { label: '部分充值', value: 8 },
  { label: '已拆单', value: 9 },
  { label: '处理中', value: 10 },
  { label: '待退款', value: 11 }
];

// 订单状态展示映射（用于列表 Tag 等）
export const ORDER_STATUS_MAP: Record<
  number,
  { type: 'success' | 'warning' | 'error' | 'info' | 'default'; text: string }
> = {
  1: { type: 'warning', text: '待支付' },
  2: { type: 'warning', text: '待充值' },
  3: { type: 'info', text: '充值中' },
  4: { type: 'success', text: '充值成功' },
  5: { type: 'error', text: '充值失败' },
  6: { type: 'info', text: '已退款' },
  7: { type: 'error', text: '已取消' },
  8: { type: 'warning', text: '部分充值' },
  9: { type: 'info', text: '已拆单' },
  10: { type: 'info', text: '处理中' },
  11: { type: 'warning', text: '待退款' }
};

/** 与后端 internal/model OrderClient* 一致 */
export const ORDER_CLIENT_EXTERNAL_API = 2;
export const ORDER_CLIENT_AGENT_MANUAL = 8;

/** 订单列表「来源」列：手动下单 / 通道或平台名 / 默认 API */
export function formatOrderSource(row: { client?: number; platform_name?: string | null }): string {
  const c = Number(row.client);
  if (c === ORDER_CLIENT_AGENT_MANUAL) return '手动下单';
  const name = row.platform_name;
  if (name != null && String(name).trim() !== '') return String(name);
  return 'API下单';
}
