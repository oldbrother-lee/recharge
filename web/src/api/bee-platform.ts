import { request } from '@/service/request';

// 蜜蜂平台用户报价库存信息接口
export interface BeeUserQuoteStockInfo {
  id: number;
  user_quote_payment: string;
  usable_stock: number;
  user_quote_discount: any;
  prov_limit_type: number;
  user_quote_type: number;
  external_code_link_type: number;
}

// 蜜蜂平台用户报价库存省份信息
export interface BeeUserQuoteStockProvInfo {
  id: number;
  quote_id: number;
  goods_id: number;
  prov: string;
  prov_id: number;
  user_quote_payment: string;
  user_quote_discount: number;
  external_code: string;
  status: number;
  last_traded_price: any;
  pin_yin: string;
}

// 蜜蜂平台商品信息接口
export interface BeeProduct {
  goods_id: number;
  goods_name: string;
  goods_type: number;
  status: number;
  user_quote_payment: number;
  prov_limit_type: number;
  user_quote_type: number;
  external_code_link_type: number;
  external_code: string;
  prov_info: BeeProvince[];
  user_quote_stock_info?: BeeUserQuoteStockInfo | null;
  user_quote_stock_prov_info?: BeeUserQuoteStockProvInfo[];
}

// 蜜蜂平台省份信息接口
export interface BeeProvince {
  prov: string;
  user_quote_payment: number;
  external_code: string;
  status: number;
}

// 蜜蜂平台商品列表响应接口
export interface BeeProductListResponse {
  code: number;
  msg: string;
  data: {
    list: BeeProduct[];
    total: number;
  };
}

// 更新商品价格请求接口
export interface BeeUpdatePriceRequest {
  goods_id: number;
  status: number;
  prov_limit_type: number;
  user_quote_type: number;
  external_code_link_type: number;
  user_quote_payment: number;
  external_code: string;
  prov_info: BeeProvince[];
}

// 更新省份配置请求接口
export interface BeeUpdateProvinceRequest {
  goods_id: number;
  provs: string[];
}

export function getBeeProductList(accountId: number, params: any) {
  return request({ url: `/platform/bee/accounts/${accountId}/products`, method: 'get', params: params });
}

export function updateBeeProductPrice(accountId: number, data: BeeUpdatePriceRequest) {
  return request({ url: `/platform/bee/accounts/${accountId}/products/price`, method: 'put', data });
}

export function updateBeeProductProvince(accountId: number, data: BeeUpdateProvinceRequest) {
  return request({ url: `/platform/bee/accounts/${accountId}/products/province`, method: 'put', data });
}