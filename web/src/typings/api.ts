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

export interface Order {
  id: number;
  order_number: string;
  out_trade_num: string;
  mobile: string;
  total_price: number;
  status: string;
  client: number;
  created_at: string;
  remark?: string;
  platform?: string;
}

export interface OrderListResponse {
  list: Order[];
  total: number;
}

export interface UserInfo {
  userId: string;
  userName: string;
  roles: string[];
  buttons: string[];
} 