package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	httpclient "github.com/dailyhot/api/internal/http"
	"github.com/dailyhot/api/internal/logger"
	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// DouyinHandler 抖音热点处理器
type DouyinHandler struct {
	fetcher *service.Fetcher
}

// NewDouyinHandler 创建抖音处理器
func NewDouyinHandler(fetcher *service.Fetcher) *DouyinHandler {
	return &DouyinHandler{
		fetcher: fetcher,
	}
}

const (
	douyinBaseURL    = "https://www.douyin.com/"
	douyinCookieURL  = "https://www.douyin.com/passport/general/login_guiding_strategy/?aid=6383"
	douyinHotListURL = "https://www.douyin.com/aweme/v1/web/hot/search/list/?device_platform=webapp&aid=6383&channel=channel_pc_web&detail_list=1"
)

var errPassportCookieNotFound = errors.New("passport_csrf_token not found")

// GetPath 获取路由路径
func (h *DouyinHandler) GetPath() string {
	return "/douyin"
}

// Handle 处理请求
func (h *DouyinHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchDouyinHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"douyin",                  // name: 平台调用名称
		"抖音",                      // title: 平台显示名称
		"热点榜",                     // type: 榜单类型
		"发现最新最热的抖音内容",             // description: 平台描述
		"https://www.douyin.com/", // link: 官方链接
		nil,                       // params: 无特殊参数映射
		data,                      // data: 热榜数据
		!noCache,                  // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchDouyinHot 从抖音 API 获取热点数据
func (h *DouyinHandler) fetchDouyinHot(ctx context.Context) ([]models.HotData, error) {
	// 1. 先获取临时 Cookie
	cookieHeader, err := h.getDouyinCookie(ctx)
	if err != nil {
		// 降级处理: 记录警告,继续尝试无 Cookie 的请求
		logger.Warn("获取抖音 Cookie 失败, 将尝试无 Cookie 请求", zap.Error(err))
		cookieHeader = ""
	}

	// 2. 请求热榜数据
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"Referer":         douyinBaseURL,
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}
	if cookieHeader != "" {
		headers["Cookie"] = cookieHeader
	}

	body, err := httpClient.Get(douyinHotListURL, headers)
	if err != nil {
		if cookieHeader != "" {
			logger.Warn("携带 Cookie 请求抖音失败, 将尝试不带 Cookie", zap.Error(err))
			delete(headers, "Cookie")
			body, err = httpClient.Get(douyinHotListURL, headers)
		}
		if err != nil {
			return nil, fmt.Errorf("请求抖音 API 失败: %w", err)
		}
	}

	// 3. 解析响应
	var apiResp DouyinAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析抖音响应失败: %w", err)
	}

	// 4. 转换为统一格式
	return h.transformData(apiResp.Data.WordList), nil
}

// getDouyinCookie 获取抖音临时 Cookie
// 对标 TypeScript 版本的 getDyCookies() 函数，确保兼容性
func (h *DouyinHandler) getDouyinCookie(ctx context.Context) (string, error) {
	_ = ctx // 当前 HTTP 客户端不支持 context 传递,占位以便后续扩展
	httpClient := h.fetcher.GetHTTPClient()

	cookies := make(map[string]string)
	baseHeaders := map[string]string{
		"Referer":         douyinBaseURL,
		"Origin":          douyinBaseURL,
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}

	// 第一次尝试直接获取 passport_csrf_token
	if tokenHeader, setCookies, err := h.requestPassportCookie(httpClient, baseHeaders, cookies); err == nil {
		return tokenHeader, nil
	} else if !errors.Is(err, errPassportCookieNotFound) {
		return "", err
	} else if len(setCookies) > 0 {
		logger.Warn("首次获取抖音 Cookie 未得到 passport_csrf_token",
			zap.Strings("set_cookie", setCookies),
		)
	}

	// 兜底: 访问主页尝试拿到 ttwid 等基础 Cookie 后再请求一次
	if _, homeErr := h.prefetchDouyinHome(httpClient, cookies); homeErr != nil {
		logger.Warn("预热抖音主页 Cookie 失败, 将继续尝试", zap.Error(homeErr))
	}

	tokenHeader, setCookies, err := h.requestPassportCookie(httpClient, baseHeaders, cookies)
	if err == nil {
		return tokenHeader, nil
	}

	if errors.Is(err, errPassportCookieNotFound) {
		debugInfo := buildSetCookieDebug(setCookies)
		return "", fmt.Errorf(
			"未找到 passport_csrf_token (共 %d 个 Set-Cookie 响应头，均不包含该字段)\n%s",
			len(setCookies),
			debugInfo,
		)
	}

	return "", err
}

// requestPassportCookie 请求登录策略接口,尝试获取 passport_csrf_token
func (h *DouyinHandler) requestPassportCookie(client *httpclient.Client, baseHeaders map[string]string, cookies map[string]string) (string, []string, error) {
	headers := make(map[string]string, len(baseHeaders)+1)
	for k, v := range baseHeaders {
		headers[k] = v
	}
	if len(cookies) > 0 {
		headers["Cookie"] = buildCookieHeader(cookies)
	}

	resp, err := client.GetWithResponse(douyinCookieURL, headers)
	if err != nil {
		return "", nil, fmt.Errorf("发起 Cookie 请求失败: %w", err)
	}

	setCookies := collectCookiesFromResponse(resp, cookies)

	if token := strings.TrimSpace(cookies["passport_csrf_token"]); token != "" {
		return buildCookieHeader(cookies), setCookies, nil
	}

	return "", setCookies, errPassportCookieNotFound
}

// prefetchDouyinHome 访问抖音首页,获取 ttwid 等基础 Cookie
func (h *DouyinHandler) prefetchDouyinHome(client *httpclient.Client, cookies map[string]string) ([]string, error) {
	headers := map[string]string{
		"Referer":         douyinBaseURL,
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}
	if len(cookies) > 0 {
		headers["Cookie"] = buildCookieHeader(cookies)
	}

	resp, err := client.GetWithResponse(douyinBaseURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求抖音主页失败: %w", err)
	}

	return collectCookiesFromResponse(resp, cookies), nil
}

// collectCookiesFromResponse 将响应中的 Cookie 收集到 map 中,并返回原始 Set-Cookie 列表
func collectCookiesFromResponse(resp *resty.Response, cookieMap map[string]string) []string {
	if resp == nil {
		return nil
	}

	// 先收集 Resty 已解析的 Cookie
	for _, c := range resp.Cookies() {
		if c == nil {
			continue
		}
		name := strings.TrimSpace(c.Name)
		value := strings.TrimSpace(c.Value)
		if name == "" || value == "" || value == "undefined" {
			continue
		}
		cookieMap[name] = value
	}

	// 再解析 Set-Cookie 响应头(确保不遗漏关键字段)
	setCookieHeaders := resp.Header().Values("Set-Cookie")
	for _, setCookie := range setCookieHeaders {
		if setCookie == "" {
			continue
		}
		parts := strings.SplitN(setCookie, ";", 2)
		if len(parts) == 0 {
			continue
		}
		nameValue := strings.TrimSpace(parts[0])
		if nameValue == "" {
			continue
		}
		nameValueParts := strings.SplitN(nameValue, "=", 2)
		if len(nameValueParts) != 2 {
			continue
		}
		name := strings.TrimSpace(nameValueParts[0])
		value := strings.TrimSpace(nameValueParts[1])
		if name == "" || value == "" || value == "undefined" {
			continue
		}
		cookieMap[name] = value
	}

	return setCookieHeaders
}

// buildSetCookieDebug 输出 Set-Cookie 调试信息
func buildSetCookieDebug(setCookies []string) string {
	if len(setCookies) == 0 {
		return "收到的 Set-Cookie 值：<无>\n"
	}

	var builder strings.Builder
	builder.WriteString("收到的 Set-Cookie 值：\n")
	for i, cookie := range setCookies {
		builder.WriteString(fmt.Sprintf("  [%d]: %s\n", i+1, cookie))
	}
	return builder.String()
}

// buildCookieHeader 将 Cookie map 构造成请求头需要的字符串
func buildCookieHeader(cookieMap map[string]string) string {
	if len(cookieMap) == 0 {
		return ""
	}

	var parts []string

	if token, ok := cookieMap["passport_csrf_token"]; ok && token != "" {
		parts = append(parts, fmt.Sprintf("passport_csrf_token=%s", token))
	}

	if defToken, ok := cookieMap["passport_csrf_token_default"]; ok && defToken != "" {
		parts = append(parts, fmt.Sprintf("passport_csrf_token_default=%s", defToken))
	}

	// 保留其他可能的 cookie，避免遗漏关键字段
	for name, value := range cookieMap {
		if name == "passport_csrf_token" || name == "passport_csrf_token_default" {
			continue
		}
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", name, value))
	}

	return strings.Join(parts, "; ")
}

// transformData 转换数据格式
func (h *DouyinHandler) transformData(items []DouyinItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		hotData := models.HotData{
			ID:        item.SentenceID,
			Title:     item.Word,
			Hot:       item.HotValue,
			URL:       fmt.Sprintf("https://www.douyin.com/hot/%s", item.SentenceID),
			MobileURL: fmt.Sprintf("https://www.douyin.com/hot/%s", item.SentenceID),
			Timestamp: item.EventTime * 1000, // 时间戳转换为毫秒级
		}

		result = append(result, hotData)
	}

	return result
}

// DouyinAPIResponse 抖音 API 响应
type DouyinAPIResponse struct {
	Data DouyinData `json:"data"`
}

// DouyinData 数据部分
type DouyinData struct {
	WordList []DouyinItem `json:"word_list"`
}

// DouyinItem 热点项
type DouyinItem struct {
	SentenceID string `json:"sentence_id"` // 热点 ID
	Word       string `json:"word"`        // 热点标题
	HotValue   int64  `json:"hot_value"`   // 热度值
	EventTime  int64  `json:"event_time"`  // 时间戳
}
