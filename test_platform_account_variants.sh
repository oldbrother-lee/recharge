#!/bin/bash

# 测试平台账号变体API的脚本

BASE_URL="http://localhost:8080/api/v1"

echo "=== 测试平台账号变体API ==="

# 1. 测试获取变体列表（需要platform_account_id参数）
echo "1. 测试获取变体列表..."
curl -s -X GET "${BASE_URL}/platform/account/variants?page=1&page_size=10&platform_account_id=1" \
  -H "Content-Type: application/json" | jq '.' || echo "请求失败或返回非JSON格式"

echo -e "\n"

# 2. 测试创建变体配置
echo "2. 测试创建变体配置..."
curl -s -X POST "${BASE_URL}/platform/account/variants" \
  -H "Content-Type: application/json" \
  -d '{
    "platform_id": 1,
    "platform_account_id": 1,
    "channel_id": 1,
    "product_id": "test_product",
    "face_values": "10,20,50",
    "min_settle_amounts": "9,18,45",
    "status": 1
  }' | jq '.' || echo "请求失败或返回非JSON格式"

echo -e "\n"

# 3. 测试获取单个变体配置（假设ID为1）
echo "3. 测试获取单个变体配置..."
curl -s -X GET "${BASE_URL}/platform/account/variants/1" \
  -H "Content-Type: application/json" | jq '.' || echo "请求失败或返回非JSON格式"

echo -e "\n"

# 4. 测试更新变体配置
echo "4. 测试更新变体配置..."
curl -s -X PUT "${BASE_URL}/platform/account/variants/1" \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "platform_id": 1,
    "platform_account_id": 1,
    "channel_id": 1,
    "product_id": "updated_product",
    "face_values": "10,20,50,100",
    "min_settle_amounts": "9,18,45,90",
    "status": 1
  }' | jq '.' || echo "请求失败或返回非JSON格式"

echo -e "\n"

# 5. 测试删除变体配置
echo "5. 测试删除变体配置..."
curl -s -X DELETE "${BASE_URL}/platform/account/variants/1" \
  -H "Content-Type: application/json" | jq '.' || echo "请求失败或返回非JSON格式"

echo -e "\n=== 测试完成 ==="