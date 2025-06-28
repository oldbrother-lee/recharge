#### 接口说明

拉取供应商品列表

**注意：请先在测试环境调通接口**

#### 接口名称

/userapi/sgd/getSupplyGoodManageList

#### 请求说明

| 请求内容     | 说明                                                         |
| :----------- | :----------------------------------------------------------- |
| 测试域名     | [http://test.shop.center.mf178.cn](http://test.shop.center.mf178.cn/) |
| 正式域名     | [https://shop.task.mf178.cn](https://shop.task.mf178.cn/)    |
| 格式         | json                                                         |
| 字符编码     | UTF-8                                                        |
| HTTP请求方式 | POST                                                         |
| 请求数限制   | 1秒1次                                                       |

#### 请求参数，其他公共参数请参考【密钥签名】

| 名称 | 类型   | 必填 | 说明                                               |
| :--- | :----- | :--- | :------------------------------------------------- |
| data | string | 是   | 上报筛选参数，参考下面 data 参数（此参数参与签名） |

#### data请求参数

| 名称          | 类型 | 必填 | 说明                                                         |
| :------------ | :--- | :--- | :----------------------------------------------------------- |
| b_id          | int  | 否   | 业务类型，可不传，不传默认查询所有业务，具体业务查看下方【业务信息】 |
| goods_id      | int  | 否   | 商品ID                                                       |
| supply_status | int  | 否   | 供货状态，可不传，不传默认查询所有状态，传0查询未供货的，传1查询已供货的，传其他查询所有的 |
| pageSize      | int  | 否   | 单页获取商品数量，默认100                                    |
| page          | int  | 否   | 获取第几页，默认第1页                                        |

##### 业务信息，请使用业务类型

| 业务类型 | 业务名称      |
| :------- | :------------ |
| 6        | 话费          |
| 7        | 油卡          |
| 3        | 权益          |
| 11       | 代下单-电影票 |
| 12       | 虚拟币        |
| 13       | 一手卡密      |

#### 请求示例

##### 上报成功请求示例

```
https://shop.task.mf178.cn/userapi/sgd/getSupplyGoodManageList
{
    "app_key":"13612274063",
    "sign":"c776ca15fef11fa187ac954daa1dfb8d",
    "timestamp":1729131765,
    "data":"{\"b_id\": 3, \"goods_id\": 0, \"supply_status\": 2, \"pageSize\": 100, \"page\": 1}"
}
```

#### 返回参数

| 名称    | 类型   | 必传 | 说明                                        |
| :------ | :----- | :--- | :------------------------------------------ |
| code    | int    | 是   | 响应状态码； 0 成功状态码；非0 错误状态码； |
| message | string | 是   | 返回结果信息                                |
| data    | array  | 是   | 列表信息，参考下面 data 参数                |

#### data 返回参数

| 名称       | 类型         | 说明                                       |
| :--------- | :----------- | :----------------------------------------- |
| goods_info | array - 多维 | 商品列表具体信息，参考下面 goods_info 参数 |
| stat_info  | array        | 列表统计信息，参考下面 stat_info 参数      |

##### goods_info 返回参数

| 名称                  | 类型   | 说明                                                         |
| :-------------------- | :----- | :----------------------------------------------------------- |
| goods_id              | int    | 商品ID                                                       |
| goods_mode            | int    | 商品模式：1非渠道管控模式，2渠道管控模式                     |
| goods_mode_text       | string | 商品模式名称                                                 |
| b_id                  | int    | 业务类型                                                     |
| b_name                | string | 业务名称                                                     |
| vender_id             | int    | 渠道ID                                                       |
| vender_name           | string | 渠道名称                                                     |
| goods_sku             | string | 规格类型                                                     |
| goods_name            | string | 规格类型名称                                                 |
| source_pack_limit_id  | int    | 商品模式goods_mode为渠道管控模式时的渠道管控ID               |
| source_limit_txt      | string | 渠道管控名称                                                 |
| spec_value_ids        | string | 规格数据                                                     |
| spec_name             | string | 规格名称                                                     |
| user_payment          | string | 平台价格                                                     |
| user_quote_mode       | int    | 报价模式：1折扣报价模式，2固定金额报价模式                   |
| user_quote_mode_text  | string | 报价模式名称                                                 |
| user_payment_range    | string | 报价区间                                                     |
| supply_status         | int    | 是否开启供应：1已供应，0未供应                               |
| need_set_prov         | bool   | 是否可以设置省份                                             |
| is_ban                | int    | 商品是否下架：1下架，0非下架                                 |
| template_price        | string | 商品模板价格                                                 |
| user_quote_stock_info | array  | 用户历史报价信息，参考下面 user_quote_stock_info 参数，可返空数组 |
| prov_code_config      | array  | 用户省份配置信息，参考下面 prov_code_config 参数，可返空数组 |
| order_limit_config    | array  | 用户终端限制信息，参考下面 order_limit_config 参数，可返空数组 |

###### user_quote_stock_info 返回参数

| 名称                    | 类型   | 说明                                               |
| :---------------------- | :----- | :------------------------------------------------- |
| user_quote_payment      | string | 用户报价价格                                       |
| usable_stock            | int    | 用户商品库存                                       |
| user_quote_discount     | string | 如果报价模式是折扣报价模式，则这个就是用户报的折扣 |
| prov_limit_type         | int    | 限制省份类型：1支持全国，2限制省份                 |
| user_quote_type         | int    | 省份报价类型：1统一报价，2按省份报价               |
| external_code_link_type | int    | 关联外部编码类型：1统一编码，2按省份编码           |

###### prov_code_config 返回参数

| 名称                | 类型   | 说明                              |
| :------------------ | :----- | :-------------------------------- |
| prov                | string | 省份名称                          |
| prov_id             | int    | 省份ID，不用管                    |
| user_quote_payment  | string | 省份报价金额                      |
| user_quote_discount | string | 省份报价折扣                      |
| external_code       | string | 省份关联外部编码                  |
| status              | string | 省份报价状态，1上架，2拉黑，3下架 |

###### order_limit_config 返回参数

| 名称             | 类型   | 说明         |
| :--------------- | :----- | :----------- |
| source_limit     | int    | 渠道管控ID   |
| source_limit_txt | string | 渠道管控名称 |
| price_limit      | string | 限价要求     |
| external_code    | string | 关联外部编码 |

##### stat_info 返回参数

| 名称           | 类型 | 说明       |
| :------------- | :--- | :--------- |
| total          | int  | 总商品数   |
| supply_total   | int  | 已供应数量 |
| unsupply_total | int  | 未供应数量 |
| page           | int  | 当前页数   |
| pageSize       | int  | 每页数量   |

#### 成功返回示例

```markdown
{
    "code": 0,
    "message": "",
    "stime": 1734420741.64412,
    "etime": 1734420747.313492,
    "data": {
        "goods_info": [
            {
                "goods_id": 10000001,
                "vender_id": 1095,
                "vender_name": "推单用户测试渠道",
                "goods_sku": "SK000177",
                "goods_name": "话费充值-移动",
                "spec_value_ids": "1,48",
                "spec_name": "移动|10",
                "user_payment": "10.000",
                "template_settle_discount": 10000,
                "user_payment_range": "9000~10000",
                "user_quote_stock_info": {
                    "id": 368,
                    "user_quote_payment": "9.000",
                    "usable_stock": 123,
                    "user_quote_discount": 9000,
                    "prov_limit_type": 2,
                    "user_quote_type": 1,
                    "external_code_link_type": 1
                },
                "source_pack_limit_id": 0,
                "source_limit_txt": "",
                "order_limit_config": {
                    "source_limit": 26,
                    "source_limit_txt": "不限制充值平台",
                    "price_limit": "1",
                    "external_code": "33",
                    "user_reject_time": null
                },
                "order_limit_config_film": {
                    "province": "",
                    "city": "",
                    "film": "",
                    "cinema": "",
                    "change_seat": "",
                    "ticket_num": ""
                },
                "supply_status": 1,
                "need_set_prov": true,
                "is_ban": 0,
                "template_price": "10.000",
                "prov_code_config": [
                    {
                        "prov": "内蒙古",
                        "prov_id": 6,
                        "user_quote_payment": "9.000",
                        "user_quote_discount": 9000,
                        "external_code": "33",
                        "status": 1
                    },
                    {
                        "prov": "北京",
                        "prov_id": 11,
                        "user_quote_payment": "9.000",
                        "user_quote_discount": 9000,
                        "external_code": "33",
                        "status": 1
                    },
                    {
                        "prov": "上海",
                        "prov_id": 13,
                        "user_quote_payment": "9.000",
                        "user_quote_discount": 9000,
                        "external_code": "33",
                        "status": 1
                    },
                    {
                        "prov": "吉林",
                        "prov_id": 16,
                        "user_quote_payment": "9.000",
                        "user_quote_discount": 9000,
                        "external_code": "33",
                        "status": 1
                    },
                    {
                        "prov": "四川",
                        "prov_id": 35,
                        "user_quote_payment": "9.000",
                        "user_quote_discount": 9000,
                        "external_code": "33",
                        "status": 1
                    },
                    {
                        "prov": "云南",
                        "prov_id": 37,
                        "user_quote_payment": "9.000",
                        "user_quote_discount": 9000,
                        "external_code": "33",
                        "status": 1
                    }
                ],
                "user_quote_mode": 1,
                "user_quote_mode_text": "折扣报价模式",
                "b_id": 6,
                "b_name": "话费",
                "goods_mode": 1,
                "goods_mode_text": "非渠道管控模式"
            },
            {
                "goods_id": 10000002,
                "vender_id": 1095,
                "vender_name": "推单用户测试渠道",
                "goods_sku": "SK000177",
                "goods_name": "话费充值-移动",
                "spec_value_ids": "1,49",
                "spec_name": "移动|20",
                "user_payment": "20.000",
                "template_settle_discount": 10000,
                "user_payment_range": "9000~10000",
                "user_quote_stock_info": {
                    "id": 620,
                    "user_quote_payment": "19.800",
                    "usable_stock": 111,
                    "user_quote_discount": 0,
                    "prov_limit_type": 2,
                    "user_quote_type": 2,
                    "external_code_link_type": 1
                },
                "source_pack_limit_id": 0,
                "source_limit_txt": "",
                "order_limit_config": {
                    "source_limit": 6,
                    "source_limit_txt": "淘宝、天猫、闲鱼",
                    "price_limit": 222,
                    "external_code": "234",
                    "user_reject_time": null
                },
                "order_limit_config_film": {
                    "province": "",
                    "city": "",
                    "film": "",
                    "cinema": "",
                    "change_seat": "",
                    "ticket_num": ""
                },
                "supply_status": 0,
                "need_set_prov": true,
                "is_ban": 0,
                "template_price": "20.000",
                "prov_code_config": [],
                "user_quote_mode": 1,
                "user_quote_mode_text": "折扣报价模式",
                "b_id": 6,
                "b_name": "话费",
                "goods_mode": 1,
                "goods_mode_text": "非渠道管控模式"
            },
            {
                "goods_id": 10000010,
                "vender_id": 1095,
                "vender_name": "推单用户测试渠道",
                "goods_sku": "SK000177",
                "goods_name": "话费充值-移动",
                "spec_value_ids": "1,50",
                "spec_name": "移动|30",
                "user_payment": "30.000",
                "template_settle_discount": 10000,
                "user_payment_range": "9000~10000",
                "user_quote_stock_info": {
                    "id": 589,
                    "user_quote_payment": "27.000",
                    "usable_stock": 10,
                    "user_quote_discount": 9000,
                    "prov_limit_type": 2,
                    "user_quote_type": 2,
                    "external_code_link_type": 2
                },
                "source_pack_limit_id": 0,
                "source_limit_txt": "",
                "order_limit_config": {
                    "source_limit": 26,
                    "source_limit_txt": "不限制充值平台",
                    "price_limit": "110",
                    "external_code": "",
                    "user_reject_time": null
                },
                "order_limit_config_film": {
                    "province": "",
                    "city": "",
                    "film": "",
                    "cinema": "",
                    "change_seat": "",
                    "ticket_num": ""
                },
                "supply_status": 0,
                "need_set_prov": true,
                "is_ban": 0,
                "template_price": "30.000",
                "prov_code_config": [
                    {
                        "prov": "广东",
                        "prov_id": 30,
                        "user_quote_payment": "27.000",
                        "user_quote_discount": 9000,
                        "external_code": "111222",
                        "status": 1
                    }
                ],
                "user_quote_mode": 1,
                "user_quote_mode_text": "折扣报价模式",
                "b_id": 6,
                "b_name": "话费",
                "goods_mode": 1,
                "goods_mode_text": "非渠道管控模式"
            }
        ],
        "stat_info": {
            "total": 120,
            "supply_total": 1,
            "unsupply_total": 119,
            "page": 1,
            "pageSize": 3
        }
    }
}
```

#### 失败返回示例

```markdown
{
    "stime": 1729156992.305196,
    "etime": 1729156992.350419,
    "code": 10002,
    "message": "签名错误",
    "data": [],
    "sucess": false
}
```

# 修改商品报价（已供应）接口

#### 接口说明

修改供应商品报价

**注意：请先在测试环境调通接口**

#### 接口名称

/userapi/sgd/editSupplyGoodManageStock

#### 请求说明

| 请求内容     | 说明                                                         |
| :----------- | :----------------------------------------------------------- |
| 测试域名     | [http://test.shop.center.mf178.cn](http://test.shop.center.mf178.cn/) |
| 正式域名     | [https://shop.task.mf178.cn](https://shop.task.mf178.cn/)    |
| 格式         | json                                                         |
| 字符编码     | UTF-8                                                        |
| HTTP请求方式 | POST                                                         |
| 请求数限制   | 1秒1次                                                       |

#### 请求参数，其他公共参数请参考【密钥签名】

| 名称 | 类型   | 必填 | 说明                                               |
| :--- | :----- | :--- | :------------------------------------------------- |
| data | string | 是   | 上报报价参数，参考下面 data 参数（此参数参与签名） |

#### data请求参数

| 名称  | 类型         | 必填 | 说明                                  |
| :---- | :----------- | :--- | :------------------------------------ |
| goods | array - 多维 | 是   | 各个商品报价信息，参考下面 goods 参数 |

##### goods请求参数

| 名称               | 类型   | 必填 | 说明                                 |
| :----------------- | :----- | :--- | :----------------------------------- |
| goods_id           | int    | 是   | 商品ID                               |
| status             | int    | 是   | 供应操作状态：1开启供应，2，关闭供应 |
| user_quote_payment | number | 否   | 商品报价：需要在商品报价范围内报价   |
| user_quote_stock   | int    | 否   | 商品库存                             |

#### 请求示例

##### 上报成功请求示例

```
https://shop.task.mf178.cn/userapi/sgd/editSupplyGoodManageStock
{
    "app_key":"13612274063",
    "sign":"c776ca15fef11fa187ac954daa1dfb8d",
    "timestamp":1729131765,
     "data":"{\"goods\": [{\"goods_id\": 10000078, \"status\": 1,\"user_quote_payment\":8000,\"user_quote_stock\":22}, {\"goods_id\": 10000001, \"status\": 1}]}"
}
```

#### 返回参数

| 名称    | 类型   | 必传 | 说明                                        |
| :------ | :----- | :--- | :------------------------------------------ |
| code    | int    | 是   | 响应状态码； 0 成功状态码；非0 错误状态码； |
| message | string | 是   | 返回结果信息                                |
| data    | array  | 是   | 处理信息，参考下面 data 参数，可返空数组    |

#### data 返回参数

| 名称        | 类型  | 说明               |
| :---------- | :---- | :----------------- |
| successMsgs | array | 修改成功的商品信息 |
| errorMsgs   | array | 修改失败的商品信息 |

#### 成功返回示例

```markdown
{
    "code": 0,
    "message": "",
    "stime": 1729137355.213657,
    "etime": 1729137355.821075,
    "data": {
        "successMsgs": {
            "10000078": "商品【10000078】报价成功并上架供应"
        },
        "errorMsgs": {
            "10000001": "商品【10000001】已被下架"
        }
    }
}
```

#### 失败返回示例

```markdown
{
    "code": 1,
    "message": "商品ID全部无效",
    "stime": 1729157745.022605,
    "etime": 1729157745.829753,
    "data": {
        "errorMsgs": {
            "10000022": "商品【10000022】不存在",
            "10000001": "商品【10000001】已被下架"
        }
    }
}
```

# 修改商品报价（话费省份报价）接口

#### 接口说明

修改供应商品报价 - 专门给话费使用的，可以进行省份报价，商品库存请用接口【修改商品报价（已供应）接口】进行上报

**注意：请先在测试环境调通接口**

#### 接口名称

/userapi/sgd/editSupplyGoodManageStockWithProv

#### 请求说明

| 请求内容     | 说明                                                         |
| :----------- | :----------------------------------------------------------- |
| 测试域名     | [http://test.shop.center.mf178.cn](http://test.shop.center.mf178.cn/) |
| 正式域名     | [https://shop.task.mf178.cn](https://shop.task.mf178.cn/)    |
| 格式         | json                                                         |
| 字符编码     | UTF-8                                                        |
| HTTP请求方式 | POST                                                         |
| 请求数限制   | 1秒1次                                                       |

#### 请求参数，其他公共参数请参考【密钥签名】

| 名称 | 类型   | 必填 | 说明                                               |
| :--- | :----- | :--- | :------------------------------------------------- |
| data | string | 是   | 上报报价参数，参考下面 data 参数（此参数参与签名） |

#### data请求参数

| 名称  | 类型         | 必填 | 说明                                  |
| :---- | :----------- | :--- | :------------------------------------ |
| goods | array - 多维 | 是   | 各个商品报价信息，参考下面 goods 参数 |

##### goods请求参数

| 名称                    | 类型         | 必填 | 说明                                                         |
| :---------------------- | :----------- | :--- | :----------------------------------------------------------- |
| goods_id                | int          | 是   | 商品ID                                                       |
| status                  | int          | 是   | 供应操作状态：1开启供应，2，关闭供应                         |
| prov_limit_type         | int          | 是   | 限制省份类型：1支持全国，2限制省份；当值传1时，user_quote_type和external_code_link_type必须都为1；当值传2时，user_quote_type和external_code_link_type必须都为2 |
| user_quote_type         | int          | 是   | 省份报价类型：1统一报价，2按省份报价                         |
| external_code_link_type | int          | 是   | 关联外部编码类型：1统一编码，2按省份编码；                   |
| user_quote_payment      | number       | 否   | 商品报价：需要在商品报价范围内报价；user_quote_type为1统一报价时，本字段必传；user_quote_type为2按省份报价时，则在省份信息里面上报 |
| external_code           | string       | 否   | 商品关联外部编码；external_code_link_type为1统一编码时，本字段必传；external_code_link_type为2按省份编码时，则在省份信息里面上报 |
| prov_info               | array - 多维 | 否   | 省份报价信息；当prov_limit_type为2限制省份时，本字段必传；字段维度参考下面 prov_info 参数；目前这个省份报价，暂时只支持一次性报价，不支持两个省份分两次报价，例如需要报上海和北京，需要在同一个请求上报上海和北京的报价信息，不能分两次上报，如果第一次只上报北京，第二次只上报上海，那么北京会被下架 |
| target                  | string       | 否   | 指定充值号段，例1524578                                      |

###### prov_info 参数

| 名称               | 类型   | 必填 | 说明                                                         |
| :----------------- | :----- | :--- | :----------------------------------------------------------- |
| prov               | string | 是   | 省份名称                                                     |
| user_quote_payment | number | 否   | 商品报价：需要在商品报价范围内报价；当user_quote_type为2按省份报价时，需要传值 |
| external_code      | string | 否   | 省份关联外部编码；当external_code_link_type为2按省份编码时，需要传值 |
| status             | string | 是   | 省份报价状态，1上架，3下架；当user_quote_type为1统一报价且external_code_link_type为1统一编码时，status默认为上架 |

#### 请求示例

##### 上报成功请求示例

```
https://shop.task.mf178.cn/userapi/sgd/editSupplyGoodManageStockWithProv
{
    "app_key":"13612274063",
    "sign":"c776ca15fef11fa187ac954daa1dfb8d",
    "timestamp":1729131765,
      "data":"{\"goods\":[{\"goods_id\":10000001,\"prov_limit_type\":1,\"user_quote_type\":1,\"external_code_link_type\":1,\"user_quote_payment\":9000,\"external_code\":\"123123\",\"status\":1},{\"goods_id\":10000002,\"prov_limit_type\":2,\"user_quote_type\":1,\"external_code_link_type\":1,\"user_quote_payment\":9000,\"external_code\":\"123123\",\"status\":1,\"prov_info\":[{\"prov\":\"内蒙古\",\"status\":1},{\"prov\":\"广东\",\"status\":3}]},{\"goods_id\":10000010,\"prov_limit_type\":2,\"user_quote_type\":2,\"external_code_link_type\":2,\"user_quote_payment\":9000,\"external_code\":\"123123\",\"status\":1,\"prov_info\":[{\"prov\":\"内蒙古\",\"user_quote_payment\":8900,\"external_code\":\"123123\",\"status\":1},{\"prov\":\"广东\",\"user_quote_payment\":9000,\"external_code\":\"111222\",\"status\":1}]}]}"
}
```

#### 返回参数

| 名称    | 类型   | 必传 | 说明                                        |
| :------ | :----- | :--- | :------------------------------------------ |
| code    | int    | 是   | 响应状态码； 0 成功状态码；非0 错误状态码； |
| message | string | 是   | 返回结果信息                                |
| data    | array  | 是   | 处理信息，参考下面 data 参数，可返空数组    |

#### data 返回参数

| 名称        | 类型  | 说明               |
| :---------- | :---- | :----------------- |
| successMsgs | array | 修改成功的商品信息 |
| errorMsgs   | array | 修改失败的商品信息 |

#### 成功返回示例

```markdown
{
    "code": 0,
    "message": "",
    "stime": 1729137355.213657,
    "etime": 1729137355.821075,
    "data": {
        "successMsgs": {
            "10000078": "商品【10000078】报价成功并上架供应"
        },
        "errorMsgs": {
            "10000001": "商品【10000001】已被下架"
        }
    }
}
```

#### 失败返回示例

```markdown
{
    "code": 1,
    "message": "商品ID全部无效",
    "stime": 1729157745.022605,
    "etime": 1729157745.829753,
    "data": {
        "errorMsgs": {
            "10000022": "商品【10000022】不存在",
            "10000001": "商品【10000001】已被下架"
        }
    }
}
```

# 修改商品省份（仅适用于话费订单）接口

#### 接口说明

修改供应商品省份 - 只能修改省份，不能报价

**注意：请先在测试环境调通接口**

#### 接口名称

/userapi/sgd/editSupplyGoodManageProvCode

#### 请求说明

| 请求内容     | 说明                                                         |
| :----------- | :----------------------------------------------------------- |
| 测试域名     | [http://test.shop.center.mf178.cn](http://test.shop.center.mf178.cn/) |
| 正式域名     | [https://shop.task.mf178.cn](https://shop.task.mf178.cn/)    |
| 格式         | json                                                         |
| 字符编码     | UTF-8                                                        |
| HTTP请求方式 | POST                                                         |
| 请求数限制   | 1秒1次                                                       |

#### 请求参数，其他公共参数请参考【密钥签名】

| 名称 | 类型   | 必填 | 说明                                               |
| :--- | :----- | :--- | :------------------------------------------------- |
| data | string | 是   | 上报报价参数，参考下面 data 参数（此参数参与签名） |

#### data请求参数

| 名称  | 类型         | 必填 | 说明                                  |
| :---- | :----------- | :--- | :------------------------------------ |
| goods | array - 多维 | 是   | 各个商品报价信息，参考下面 goods 参数 |

##### goods请求参数

| 名称     | 类型  | 必填 | 说明                                                         |
| :------- | :---- | :--- | :----------------------------------------------------------- |
| goods_id | int   | 是   | 商品ID                                                       |
| provs    | array | 是   | 设置省份集合，使用方式参考示例，只能设置下列的省份【上海、云南、全国、内蒙古、北京、台湾、吉林、四川、天津、宁夏、安徽、山东、山西、广东、广西、新疆、江苏、江西、河北、河南、浙江、海南、深圳、湖北、湖南、甘肃、福建、西藏、贵州、辽宁、重庆、陕西、青海、黑龙江】 |

#### 请求示例

##### 上报成功请求示例

```
https://shop.task.mf178.cn/userapi/sgd/editSupplyGoodManageProvCode
{
    "app_key":"13612274063",
    "sign":"c776ca15fef11fa187ac954daa1dfb8d",
    "timestamp":1729131765,
    "data": "{\"goods\":[{\"goods_id\":\"10000001\",\"provs\":[\"广东\",\"上海\"]},{\"goods_id\":\"10000002\",\"provs\":[\"全国\",\"广东\"]},{\"goods_id\":\"10000078\",\"provs\":[\"广东2\",\"上海\"]}]}"
}
```

#### 返回参数

| 名称    | 类型   | 必传 | 说明                                        |
| :------ | :----- | :--- | :------------------------------------------ |
| code    | int    | 是   | 响应状态码； 0 成功状态码；非0 错误状态码； |
| message | string | 是   | 返回结果信息                                |
| data    | array  | 是   | 处理信息，参考下面 data 参数，可返空数组    |

#### data 返回参数

| 名称        | 类型  | 说明               |
| :---------- | :---- | :----------------- |
| successMsgs | array | 修改成功的商品信息 |
| errorMsgs   | array | 修改失败的商品信息 |

#### 成功返回示例

```markdown
{
    "code": 0,
    "message": "",
    "stime": 1729158063.640588,
    "etime": 1729158064.792595,
    "data": {
        "successMsgs": {
            "10000001": "商品【10000001】省份修改[广东,上海]",
            "10000002": "商品【10000002】省份修改[全国]"
        },
        "errorMsgs": {
            "10000078": "商品【10000078】不能设置省份"
        }
    }
}
```

#### 失败返回示例

```markdown
{
    "code": 1,
    "message": "商品ID全部无效",
    "stime": 1729158123.414051,
    "etime": 1729158123.604055,
    "data": {
        "errorMsgs": []
    }
}
```