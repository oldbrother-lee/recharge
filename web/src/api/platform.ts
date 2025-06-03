import { request } from '@/service/request';

/** API响应 */
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

/** 渠道信息 */
export interface Channel {
  /** 渠道编号 */
  channelId: number;
  /** 渠道名称 */
  channelName: string;
  /** 渠道对应下的运营商信息 */
  productList: Product[];
}

/** 运营商信息 */
export interface Product {
  /** 运营商编号 */
  productId: number;
  /** 运营商名称 */
  productName: string;
}

/** 获取渠道列表 */
export function getChannelList() {
  return request({ url: '/platform/xianzhuanxia/channels', method: 'get' });
} 

