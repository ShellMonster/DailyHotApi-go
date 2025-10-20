package routes

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

// IthomeHandler IT之家处理器
type IthomeHandler struct {
	fetcher *service.Fetcher
}

// NewIthomeHandler 创建IT之家处理器
func NewIthomeHandler(fetcher *service.Fetcher) *IthomeHandler {
	return &IthomeHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *IthomeHandler) GetPath() string {
	return "/ithome"
}

// Handle 处理请求
func (h *IthomeHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取热榜数据
	data, err := h.fetchIthomeHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"ithome",                  // name: 平台调用名称
		"IT之家",                    // title: 平台显示名称
		"热榜",                      // type: 榜单类型
		"发现IT之家热门资讯",              // description: 平台描述
		"https://www.ithome.com/", // link: 官方链接
		nil,                       // params: 无参数映射
		data,                      // data: 热榜数据
		!noCache,                  // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchIthomeHot 从IT之家获取热榜数据
func (h *IthomeHandler) fetchIthomeHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://m.ithome.com/rankm/"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求IT之家失败: %w", err)
	}

	// 解析 HTML
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析 HTML
func (h *IthomeHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return result
	}

	digitPattern := regexp.MustCompile(`\d+`)

	doc.Find(".rank-box .placeholder").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".plc-title").Text())
		if title == "" {
			return
		}

		href, exists := s.Find("a").Attr("href")
		if !exists || href == "" {
			return
		}

		cover, _ := s.Find("img").Attr("data-original")

		timeText := strings.TrimSpace(s.Find("span.post-time").Text())
		timestamp := timeutil.ParseTime(timeText)

		hot := int64(0)
		if match := digitPattern.FindString(s.Find(".review-num").Text()); match != "" {
			hot, _ = strconv.ParseInt(match, 10, 64)
		}

		// 处理链接
		url := h.replaceLink(href)
		id := h.extractID(href)

		hotData := models.HotData{
			ID:        id,
			Title:     title,
			Cover:     cover,
			Hot:       hot,
			URL:       url,
			MobileURL: url,
			Timestamp: timestamp,
		}

		result = append(result, hotData)

		if len(result) >= 50 {
			return
		}
	})

	return result
}

// replaceLink 链接处理
// TypeScript 正则: /[html|live]\/(\d+)\.htm/
// 这里的 [html|live] 在 TypeScript 中表示 "html|live"（或操作）
// 在 Go 正则中，[...] 表示字符集，需要改用 (?:html|live) 来表示"或"
func (h *IthomeHandler) replaceLink(url string) string {
	// 修正正则模式：(?:html|live) 表示 "html 或 live"
	pattern := regexp.MustCompile(`(?:html|live)/(\d+)\.htm`)
	matches := pattern.FindStringSubmatch(url)
	if len(matches) > 1 {
		id := matches[1]
		// 分割 ID：前3位、后续位数
		if len(id) >= 3 {
			return fmt.Sprintf("https://www.ithome.com/0/%s/%s.htm", id[:3], id[3:])
		}
		return url
	}
	return url
}

// extractID 提取 ID
// TypeScript 正则: /[html|live]\/(\d+)\.htm/
func (h *IthomeHandler) extractID(url string) string {
	// 修正正则模式：(?:html|live) 表示 "html 或 live"
	pattern := regexp.MustCompile(`(?:html|live)/(\d+)\.htm`)
	matches := pattern.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return "0"
}
