export interface ProductType {
  id: number;
  type_name: string;
  status: number;
  sort: number;
  account_type: number;
  tishi_doc: string;
  icon: string;
}

export interface ProductCategory {
  id: number;
  name: string;
  sort: number;
  status: number;
}

export interface Product {
  id: number;
  name: string;
  description: string;
  type_id: number;
  category_id: number;
  product_type?: ProductType;
  category?: ProductCategory;
  isp: string;
  price: number;
  max_price: number;
  voucher_price: string;
  voucher_name: string;
  show_style: number;
  api_fail_style: number;
  allow_provinces: string;
  allow_cities: string;
  forbid_provinces: string;
  forbid_cities: string;
  api_delay: string;
  sort: number;
  status: number;
  api_enabled: boolean;
  is_decode: boolean;
  remark: string;
  created_at: string;
  updated_at: string;
} 