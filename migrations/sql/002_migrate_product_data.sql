-- 迁移商品分类数据
INSERT INTO product_categories (
    id,
    name,
    sort,
    status,
    created_at,
    updated_at
)
SELECT 
    id,
    cate,
    sort,
    type,
    NOW(),
    NOW()
FROM recharge.dyr_product_cate;

-- 迁移商品数据
INSERT INTO products (
    id,
    name,
    description,
    price,
    type,
    isp,
    created_at,
    status,
    sort,
    api_enabled,
    remark,
    category_id,
    operator_tag,
    max_price,
    voucher_price,
    voucher_name,
    show_style,
    api_fail_style,
    allow_provinces,
    allow_cities,
    forbid_provinces,
    forbid_cities,
    api_delay,
    grade_ids,
    api_id,
    api_param_id,
    is_decode,
    created_time,
    updated_time
)
SELECT 
    id,
    name,
    `desc`,
    price,
    type,
    isp,
    FROM_UNIXTIME(create_time),
    CASE WHEN is_del = 1 THEN 0 ELSE 1 END,
    sort,
    api_open,
    remark,
    cate_id,
    ys_tag,
    max_price,
    voucher_price,
    voucher_name,
    show_style,
    api_fail_style,
    allow_pro,
    allow_city,
    forbid_pro,
    forbid_city,
    delay_api,
    grade_ids,
    jmapi_id,
    jmapi_param_id,
    is_jiema,
    FROM_UNIXTIME(create_time),
    NOW()
FROM recharge.dyr_product
WHERE is_del = 0;

-- 迁移商品规格数据
INSERT INTO product_specs (
    product_id,
    name,
    value,
    price,
    stock,
    sort,
    created_at,
    updated_at
)
SELECT 
    product_id,
    reapi_id,
    param_id,
    0.00,
    num,
    sort,
    NOW(),
    NOW()
FROM recharge.dyr_product_api
WHERE status = 1;

-- 迁移会员等级数据
INSERT INTO member_grades (
    id,
    name,
    grade_type,
    sort,
    created_at,
    updated_at
)
SELECT 
    id,
    grade_name,
    grade_type,
    sort,
    NOW(),
    NOW()
FROM recharge.dyr_customer_grade
WHERE is_agent = 0;

-- 迁移商品会员价格数据
INSERT INTO product_grade_prices (
    product_id,
    grade_id,
    price,
    created_at,
    updated_at
)
SELECT 
    product_id,
    grade_id,
    ranges,
    NOW(),
    NOW()
FROM recharge.dyr_customer_grade_price; 