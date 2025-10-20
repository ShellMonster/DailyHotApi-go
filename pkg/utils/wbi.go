package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// WBI 签名相关常量
// B站的反爬虫机制,需要对请求参数进行特殊签名
var (
	// mixinKeyEncTab 混淆表
	// 这是 B站用来混淆密钥的固定映射表
	mixinKeyEncTab = []int{
		46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
		33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
		61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
		36, 20, 34, 44, 52,
	}

	// 固定的 img_key 和 sub_key (从 B站 API 获取,这里使用示例值)
	// 实际使用时应该从 B站 nav 接口动态获取
	// imgKey = "7cd084941338484aae1ad9425b84077c" // 示例,实际需要动态获取
	// subKey = "4932caff0ff746eab6f01bf08b70ac45" // 示例,实际需要动态获取
)

// GetMixinKey 获取混淆后的密钥
// orig: 原始密钥(imgKey 或 subKey)
// 返回: 混淆后的 32 位密钥
func GetMixinKey(orig string) string {
	// 使用 mixinKeyEncTab 对原始密钥进行重排
	var mixinKey []byte
	for _, i := range mixinKeyEncTab {
		if i < len(orig) {
			mixinKey = append(mixinKey, orig[i])
		}
	}

	// 取前 32 位
	if len(mixinKey) > 32 {
		mixinKey = mixinKey[:32]
	}

	return string(mixinKey)
}

// EncodeWBI 对请求参数进行 WBI 签名
// params: 原始请求参数
// imgKey, subKey: 从 B站 API 获取的密钥
// 返回: 签名后的参数字符串
func EncodeWBI(params map[string]string, imgKey, subKey string) string {
	// 1. 获取混淆后的 mixin_key
	mixinKey := GetMixinKey(imgKey + subKey)

	// 2. 添加当前时间戳(秒)
	params["wts"] = strconv.FormatInt(time.Now().Unix(), 10)

	// 3. 对参数按 key 排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 4. 拼接参数字符串
	query := ""
	for i, key := range keys {
		// 过滤特殊字符(B站要求)
		value := filterValue(params[key])

		if i > 0 {
			query += "&"
		}
		query += fmt.Sprintf("%s=%s", key, value)
	}

	// 5. 计算 w_rid (MD5 签名)
	query += mixinKey
	hash := md5.Sum([]byte(query))
	wRid := hex.EncodeToString(hash[:])

	// 6. 返回最终的查询参数(包含 w_rid)
	finalQuery := ""
	for i, key := range keys {
		if i > 0 {
			finalQuery += "&"
		}
		finalQuery += fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(params[key]))
	}
	finalQuery += fmt.Sprintf("&w_rid=%s", wRid)

	return finalQuery
}

// filterValue 过滤参数值中的特殊字符
// B站要求移除一些特殊字符,防止注入攻击
func filterValue(value string) string {
	// 移除: !'()*
	unwantedChars := "!'()*"
	result := ""
	for _, char := range value {
		if !contains(unwantedChars, char) {
			result += string(char)
		}
	}
	return result
}

// contains 检查字符串是否包含某个字符
func contains(s string, char rune) bool {
	for _, c := range s {
		if c == char {
			return true
		}
	}
	return false
}

// GetNavInfo 从 B站获取 nav 信息(包含 img_key 和 sub_key)
// 这个函数需要实际请求 B站 API
// 返回: imgKey, subKey, error
func GetNavInfo() (string, string, error) {
	// TODO: 实际实现需要请求 B站 API
	// https://api.bilibili.com/x/web-interface/nav
	// 从返回的 wbi_img.img_url 和 wbi_img.sub_url 中提取密钥

	// 临时使用固定值(实际应该动态获取)
	imgKey := "7cd084941338484aae1ad9425b84077c"
	subKey := "4932caff0ff746eab6f01bf08b70ac45"

	return imgKey, subKey, nil
}
