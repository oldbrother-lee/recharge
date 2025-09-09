# 签名规则

### sign 签名生成规则

步骤

1.

将请求的数据按照键名进行升序排序。

2.

将参数、值进行拼接。拼接格式为参数名称=参数值....。

3.

拼接完的字符串再和商户的app_secret进行拼接，拼接完成之后整体进行md5处理，最后将md5的结果进行小写转换即可。

### 示例

例如提交到接口的参数为(排除 sign 参数)：
`order_no=6512BB25D61657EF44&app_key=13812345678&timestamp=2023-09-26 11:25:00`
根据参数名称进行升序排序
`app_key=13812345678&order_no=6512BB25D61657EF44&timestamp=2023-09-26 11:25:00`
在最后加上 appSecret 的值，例如是：squ2clksvdmquzurk4tioblcosuzddsh
`app_key=13812345678&order_no=6512BB25D61657EF44&timestamp=2023-09-26 11:25:00squ2clksvdmquzurk4tioblcosuzddsh`
MD5 加密（UTF-8 编码，结果转为 32 位长度的 16 进制小写字符串）
`8b30c1909257e5ab7b932e1d164ec954`
将该值作为 sign 的值 追加到请求参数中
`{"order_no":"6512BB25D61657EF44","timestamp":"2023-09-26 11:25:00","app_key":"13812345678","sign":"8b30c1909257e5ab7b932e1d164ec954"}`



### golang签名生成示例代码：

```
package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
)

func main() {
	params := make(map[string]string)
	params["app_key"] = "13812345678"
	params["order_no"] = "6512BB25D61657EF44"
	params["timestamp"] = "2023-09-26 11:25:00"
	//加密秘钥
	appSecret := "squ2clksvdmquzurk4tioblcosuzddsh"
	sign := getSign(params, appSecret)
	fmt.Printf("sign is: %s\n", sign)
}

func getSign(params map[string]string, appSecret string) string {
	var dataParams string
	//ksort
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	//拼接
	for _, k := range keys {
		fmt.Println("key:", k, "Value:", params[k])
		dataParams = dataParams + k + "=" + params[k] + "&"
	}
	//去掉最后一个&符拼接appSecret 输出 app_key=64a6285f23c5a&customer_order_no=16897367282&product_code=JDEMWh9Ij31oOWpiy2X
	signString := dataParams[0:len(dataParams)-1] + appSecret
	fmt.Println("sign string:" + signString)
	//md5加密
	d := []byte(signString)
	m := md5.New()
	m.Write(d)
	sign := hex.EncodeToString(m.Sum(nil))
	return sign
}
```

# product_id运营商

| **运营商编码** | **运营商编码说明** |
| :------------- | :----------------- |
| 415            | 移动               |
| 418            | 联通               |
| 419            | 电信               |

### **province_id地区**

| **地区编码** | **地区编码说明** |
| :----------- | :--------------- |
| 11           | 北京             |
| 12           | 天津             |
| 13           | 河北             |
| 14           | 山西             |
| 15           | 内蒙古           |
| 21           | 辽宁             |
| 22           | 吉林             |
| 23           | 黑龙江           |
| 31           | 上海             |
| 32           | 江苏             |
| 33           | 浙江             |
| 34           | 安徽             |
| 35           | 福建             |
| 36           | 江西             |
| 37           | 山东             |
| 41           | 河南             |
| 42           | 湖北             |
| 43           | 湖南             |
| 44           | 广东             |
| 45           | 广西             |
| 46           | 海南             |
| 50           | 重庆             |
| 51           | 四川             |
| 52           | 贵州             |
| 53           | 云南             |
| 54           | 西藏             |
| 61           | 陕西             |
| 62           | 甘肃             |
| 63           | 青海             |
| 64           | 宁夏             |
| 65           | 新疆             |
|              |                  |

# 对接注意事项

### **充值超时**

**超过订单充值超时时间后，订单将会自动失败，请勿继续充值**

### **开启推单**

商户后台配置推单地址后开启推单
1.配置推单地址

2.开启推单



# 订单状态编码对照表

### **code请求响应编码**

| **响应编码** | **响应编码说明**  |
| :----------- | :---------------- |
| 0            | 请求成功          |
| 5003         | 更新数据失败      |
| 10000        | 失败,服务器错误   |
| 10100        | 失败,请求参数错误 |

### **status订单状态码**

| **订单状态码** | **订单状态码说明** |
| :------------- | :----------------- |
| 2              | 处理中             |
| 3              | 结果上报异常       |
| 4              | 订单失败           |
| 5              | 订单成功           |

### **settle_status结算状态码**

| **结算状态码** | **结算状态码说明** |
| :------------- | :----------------- |
| 1              | 结算中             |
| 2              | 已结算             |
| 3              | 待结算             |

# 推送订单 （需要我们提供接口）

## 请求参数

Body 参数application/json

id

integer 

平台订单号

必需

product_id

integer 

必需

取值详见编码对照表, product_id运营商

province_id

string 

必需

取值详见编码对照表, province_id归属地

chan_pro_code

string 

商户自定义产品编码

必需

market_price

integer 

充值金额(元)

必需

account

string 

充值账号

必需

settle_money

string 

结算金额(元)

必需

settle_status

integer 

必需

取值详见订单状态编码对照表, 结算状态

timeout

integer 

必需

超时时间戳，超过此时间上报结果，订单会自动置为失败

status

integer 

必需

取值详见订单状态编码对照表, status订单状态

app_key

string 

商户号

必需

sign

string 

签名

必需

示例

```
{
    "id": 202576594079,
    "product_id": 418,
    "province_id": "13",
    "chan_pro_code": "",
    "market_price": 50,
    "account": "18631791255",
    "settle_money": "49.0000",
    "settle_status": 3,
    "timeout": 1754303300,
    "status": 2,
    "app_key": "67c8f8580e958",
    "sign": "110c07b43af89e88536d9d1003808322"
}
```

## 请求示例代码





```
package main

import (
   "fmt"
   "strings"
   "net/http"
   "io/ioutil"
)

func main() {

   url := "http://api.foreverhappy.cn//%E5%95%86%E6%88%B7%E5%90%8E%E5%8F%B0%E9%85%8D%E7%BD%AE%E7%9A%84%E6%8E%A8%E9%80%81%E5%9C%B0%E5%9D%80"
   method := "POST"

   payload := strings.NewReader(`{
    "id": 202576594079,
    "product_id": 418,
    "province_id": "13",
    "chan_pro_code": "",
    "market_price": 50,
    "account": "18631791255",
    "settle_money": "49.0000",
    "settle_status": 3,
    "timeout": 1754303300,
    "status": 2,
    "app_key": "67c8f8580e958",
    "sign": "110c07b43af89e88536d9d1003808322"
}`)

   client := &http.Client {
   }
   req, err := http.NewRequest(method, url, payload)

   if err != nil {
      fmt.Println(err)
      return
   }
   req.Header.Add("Content-Type", "application/json")

   res, err := client.Do(req)
   if err != nil {
      fmt.Println(err)
      return
   }
   defer res.Body.Close()

   body, err := ioutil.ReadAll(res.Body)
   if err != nil {
      fmt.Println(err)
      return
   }
   fmt.Println(string(body))
}
```

## 返回响应



🟢200成功

application/json

result

string 

推送结果(接收成功返回success,接收失败或者拒绝接收返回fail)

必需

user_no

string 

商户自定义单号

可选

示例

拒绝接单示例

成功示例

```
{
    "result": "fail"
}
```

# 3.充值结果上报 (网络异常时，必需再次上报结果)

## OpenAPI Specification

```yaml
openapi: 3.0.1
info:
  title: ''
  description: ''
  version: 1.0.0
paths:
  /api/v1/helpOrder/report:
    post:
      summary: 3.充值结果上报 (网络异常时，必需再次上报结果)
      deprecated: false
      description: ''
      tags:
        - 帮充Api文档
      parameters: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                app_key:
                  type: string
                  description: 商户号
                id:
                  type: string
                  description: 平台订单号
                status:
                  type: integer
                  description: 取值详见订单状态编码对照表, status订单状态,只能上报成功和失败
                error_msg:
                  type: string
                  description: 失败原因
                voucher_text:
                  type: string
                  description: 充值凭证文本
                voucher_base64:
                  type: string
                  description: 如果上报充值凭证图片，请上报Base64图片
                timestamp:
                  type: string
                  description: 请求时间
                sign:
                  type: string
                  description: 签名
              required:
                - app_key
                - id
                - status
                - error_msg
                - timestamp
                - sign
              x-apifox-orders:
                - app_key
                - id
                - status
                - error_msg
                - voucher_text
                - voucher_base64
                - timestamp
                - sign
            example:
              app_key: 67c8f8580e958
              id: '1749711924'
              status: 5
              error_msg: 失败原因
              voucher_text: 充值凭证
              voucher_base64: 如果上报凭证图片，请上报Base64图片
              timestamp: '2025-02-17 18:22:22'
              sign: 5852ab76aa22a3d1a2d9b916099c0bb8
      responses:
        '200':
          description: ''
          content:
            application/json:
              schema:
                type: object
                properties:
                  code:
                    type: integer
                    title: 取值详见订单状态编码对照表，code请求响应编码
                  data:
                    type: object
                    properties:
                      id:
                        type: integer
                        title: 平台订单号
                      product_id:
                        type: integer
                        title: 取值详见编码对照表, product_id运营商
                      market_price:
                        type: integer
                        title: 充值金额(元)
                      account:
                        type: string
                        title: 充值账号
                      settle_money:
                        type: integer
                        title: 结算金额(元)
                      settle_status:
                        type: integer
                        title: 取值详见订单状态编码对照表, settle_status结算状态
                      status:
                        type: integer
                        title: 取值详见订单状态编码对照表, status订单状态
                      allow_channel_type:
                        type: string
                        title: 取值详见编码对照表, channel充值渠道
                      timeout:
                        type: integer
                        title: 超时时间戳，超过此时间上报结果，订单会自动置为失败
                    required:
                      - id
                      - product_id
                      - market_price
                      - account
                      - settle_money
                      - settle_status
                      - timeout
                      - status
                      - allow_channel_type
                    x-apifox-orders:
                      - id
                      - product_id
                      - market_price
                      - account
                      - settle_money
                      - settle_status
                      - timeout
                      - status
                      - allow_channel_type
                  msg:
                    type: string
                required:
                  - code
                  - data
                  - msg
                x-apifox-orders:
                  - code
                  - data
                  - msg
              example:
                code: 0
                data:
                  id: 1749711924
                  product_id: 415
                  market_price: 50
                  account: '11111111111'
                  settle_money: 50
                  settle_status: 3
                  timeout: 1749713724
                  status: 5
                  allow_channel_type: jdm,jdweb,pddm,pddapp,pddwx,yzfapp
                msg: success
          headers: {}
          x-apifox-name: 成功
      security: []
      x-apifox-folder: 帮充Api文档
      x-apifox-status: released
      x-run-in-apifox: https://app.apifox.com/web/project/6570761/apis/api-308241779-run
components:
  schemas: {}
  securitySchemes: {}
servers:
  - url: http://api.foreverhappy.cn/
    description: 正式环境
security: []

```