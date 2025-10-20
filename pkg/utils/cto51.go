package utils

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// Get51CTOToken 获取51CTO API Token
// Token有效期较长,建议缓存使用
func Get51CTOToken() (string, error) {
	url := "https://api-media.51cto.com/api/token-get"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析Token响应失败: %w", err)
	}

	return result.Data.Data.Token, nil
}

// Sign51CTO 生成51CTO API签名
// requestPath: 请求路径,如 "index/index/recommend"
// params: 请求参数
// timestamp: 时间戳(毫秒)
// token: API Token
func Sign51CTO(requestPath string, params map[string]interface{}, timestamp int64, token string) string {
	// 1. 添加timestamp和token到参数中
	params["timestamp"] = timestamp
	params["token"] = token

	// 2. 对参数key排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 将排序后的keys转为字符串
	keysStr := ""
	for _, k := range keys {
		keysStr += k
	}

	// 4. 计算签名
	// sign = MD5(MD5(requestPath) + MD5(sortedKeys + MD5(token) + timestamp))
	pathMD5 := fmt.Sprintf("%x", md5.Sum([]byte(requestPath)))
	tokenMD5 := fmt.Sprintf("%x", md5.Sum([]byte(token)))
	middlePart := keysStr + tokenMD5 + strconv.FormatInt(timestamp, 10)
	middleMD5 := fmt.Sprintf("%x", md5.Sum([]byte(middlePart)))
	finalStr := pathMD5 + middleMD5
	sign := fmt.Sprintf("%x", md5.Sum([]byte(finalStr)))

	return sign
}
