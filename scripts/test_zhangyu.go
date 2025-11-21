package main

import (
	"fmt"
	"time"
)

// 章鱼平台测试程序
func main() {
	// 配置信息（需要联系运营人员获取）
	devID := "1861"            // 开发者ID
	rc4Key := "wLkJMEoC"       // RC4密钥（需要联系运营人员提供）
	username := "18635637493b" // 用户名
	password := "112233"       // 密码
	flag := "dxapp2665"        // 渠道编码

	fmt.Println("=== 章鱼平台API测试 ===")
	fmt.Println("注意：需要配置正确的devID、rc4Key、username和password")
	fmt.Println()

	// 1. 登录
	fmt.Println("=== 1. 登录 ===")
	token, err := ZYLogin(devID, rc4Key, username, password)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		fmt.Println("\n提示：请检查配置信息是否正确")
		return
	}
	fmt.Printf("登录成功，获取到token: %s\n", token)

	// 2. 获取订单（循环获取，每秒一次）
	fmt.Println("\n=== 2. 循环获取订单（每秒一次）===")
	fmt.Println("按 Ctrl+C 停止程序")

	counter := 1
	for {
		fmt.Printf("\n--- 第 %d 次获取订单 ---\n", counter)
		fmt.Printf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))

		orderResp, err := ZYGetOrder(devID, rc4Key, token, flag, "0", "100", "广西")
		if err != nil {
			fmt.Printf("获取订单失败: %v\n", err)
		} else {
			if orderResp.Ret == 0 {
				fmt.Printf("获取订单成功!\n")
				fmt.Printf("订单ID: %d\n", orderResp.Data.ID)
				fmt.Printf("充值面值: %d元\n", orderResp.Data.Amount)
				fmt.Printf("充值号码: %s\n", orderResp.Data.Mobile)
				fmt.Printf("运营商: %s\n", orderResp.Data.Operator)
				fmt.Printf("运营商ID: %d\n", orderResp.Data.OperatorID)
				fmt.Printf("省份: %s\n", orderResp.Data.Prov)
				fmt.Printf("超时时间: %d\n", orderResp.Data.Timeout)

				// 3. 上报订单
				fmt.Println("\n[上报] 开始上报订单...")
				orderCreateTime := time.Now().Format("2006-01-02 15:04:05")
				reportResp, err := ZYReportOrder(devID, rc4Key, token, flag,
					fmt.Sprintf("%d", orderResp.Data.ID), "2", "支付失败", "", "", orderCreateTime)
				if err != nil {
					fmt.Printf("[上报] 失败: %v\n", err)
				} else {
					fmt.Printf("[上报] 成功!\n")
					fmt.Printf("[上报] 返回码: %d\n", reportResp.Ret)
					fmt.Printf("[上报] 返回消息: %s\n", reportResp.Msg)
				}
			} else {
				fmt.Printf("获取订单失败: %s\n", orderResp.Msg)
			}
		}
		//获取到订单退出循环,订单 id不为 0 则退出循环
		if orderResp.Data.ID != 0 {
			break
		}
		counter++
		time.Sleep(1 * time.Second)
	}
}
