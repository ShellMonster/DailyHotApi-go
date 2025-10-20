package utils

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"
)

// GetRandomDeviceID 获取随机的设备ID
func GetRandomDeviceID() string {
	lengths := []int{10, 6, 6, 6, 14}
	parts := make([]string, len(lengths))

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i, length := range lengths {
		b := make([]byte, length)
		for j := range b {
			b[j] = charset[rng.Intn(len(charset))]
		}
		parts[i] = string(b)
	}

	return fmt.Sprintf("%s-%s-%s-%s-%s", parts[0], parts[1], parts[2], parts[3], parts[4])
}

// GetAppToken 获取酷安APP Token
func GetAppToken() string {
	deviceID := GetRandomDeviceID()
	now := time.Now().Unix()
	hexNow := fmt.Sprintf("0x%x", now)

	// MD5(now)
	md5Now := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", now))))

	// 构建签名字符串
	s := fmt.Sprintf("token://com.coolapk.market/c67ef5943784d09750dcfbb31020f0ab?%s$%s&com.coolapk.market",
		md5Now, deviceID)

	// Base64编码后再MD5
	sBase64 := base64.StdEncoding.EncodeToString([]byte(s))
	md5S := fmt.Sprintf("%x", md5.Sum([]byte(sBase64)))

	// 组合token
	token := fmt.Sprintf("%s%s%s", md5S, deviceID, hexNow)
	return token
}

// GenCoolapkHeaders 生成酷安请求头
func GenCoolapkHeaders() map[string]string {
	return map[string]string{
		"X-Requested-With": "XMLHttpRequest",
		"X-App-Id":         "com.coolapk.market",
		"X-App-Token":      GetAppToken(),
		"X-Sdk-Int":        "29",
		"X-Sdk-Locale":     "zh-CN",
		"X-App-Version":    "11.0",
		"X-Api-Version":    "11",
		"X-App-Code":       "2101202",
		"User-Agent":       "Mozilla/5.0 (Linux; Android 10; Mi 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.5563.15 Mobile Safari/537.36",
	}
}
