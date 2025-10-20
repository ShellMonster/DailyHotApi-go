package routes

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// IthomeXijiayiHandler IT之家喜加一处理器
type IthomeXijiayiHandler struct {
	fetcher *service.Fetcher
}

// NewIthomeXijiayiHandler 创建IT之家喜加一处理器
func NewIthomeXijiayiHandler(fetcher *service.Fetcher) *IthomeXijiayiHandler {
	return &IthomeXijiayiHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *IthomeXijiayiHandler) GetPath() string {
	return "/ithome-xijiayi"
}

// Handle 处理请求
func (h *IthomeXijiayiHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchIthomeXijiayiHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"ithome_xijiayi",
		"IT之家「喜加一」",
		"最新动态",
		"IT之家「喜加一」最新动态列表",
		"https://www.ithome.com/zt/xijiayi",
		nil,
		data,
		!noCache,
	))
}

// fetchIthomeXijiayiHot 从IT之家获取喜加一数据
func (h *IthomeXijiayiHandler) fetchIthomeXijiayiHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.ithome.com/zt/xijiayi"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("请求IT之家失败: %w", err)
	}

	// 解析HTML
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析HTML
func (h *IthomeXijiayiHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	// 匹配 .newslist li 项
	liPattern := regexp.MustCompile(`(?s)<li[^>]*>(.*?)</li>`)
	liMatches := liPattern.FindAllStringSubmatch(html, -1)

	for _, liMatch := range liMatches {
		if len(liMatch) < 2 {
			continue
		}

		liHTML := liMatch[1]

		// 必须包含 a 标签才是有效项
		if !strings.Contains(liHTML, "<a") {
			continue
		}

		// 提取链接
		hrefPattern := regexp.MustCompile(`<a[^>]+href="([^"]+)"`)
		hrefMatches := hrefPattern.FindStringSubmatch(liHTML)
		href := ""
		if len(hrefMatches) > 1 {
			href = hrefMatches[1]
		}

		// 提取标题
		titlePattern := regexp.MustCompile(`<h2[^>]*>([^<]+)</h2>`)
		titleMatches := titlePattern.FindStringSubmatch(liHTML)
		title := ""
		if len(titleMatches) > 1 {
			title = strings.TrimSpace(titleMatches[1])
		}

		// 提取描述
		descPattern := regexp.MustCompile(`<p[^>]*>([^<]+)</p>`)
		descMatches := descPattern.FindStringSubmatch(liHTML)
		desc := ""
		if len(descMatches) > 1 {
			desc = strings.TrimSpace(descMatches[1])
		}

		// 提取封面
		coverPattern := regexp.MustCompile(`<img[^>]+data-original="([^"]+)"`)
		coverMatches := coverPattern.FindStringSubmatch(liHTML)
		cover := ""
		if len(coverMatches) > 1 {
			cover = coverMatches[1]
		}

		// 提取时间
		timePattern := regexp.MustCompile(`<span[^>]+class="time"[^>]*>'([^']+)'`)
		timeMatches := timePattern.FindStringSubmatch(liHTML)
		timestamp := ""
		if len(timeMatches) > 1 {
			timestamp = h.parseTime(timeMatches[1])
		}

		// 提取评论数（热度）
		commentPattern := regexp.MustCompile(`<span[^>]+class="comment"[^>]*>(\d+)`)
		commentMatches := commentPattern.FindStringSubmatch(liHTML)
		hot := int64(0)
		if len(commentMatches) > 1 {
			hot, _ = strconv.ParseInt(commentMatches[1], 10, 64)
		}

		// 转换链接和ID
		id, mobileURL := h.replaceLink(href)

		hotData := models.HotData{
			ID:        id,
			Title:     title,
			Desc:      desc,
			Cover:     cover,
			Hot:       hot,
			Timestamp: timestamp,
			URL:       href,
			MobileURL: mobileURL,
		}

		result = append(result, hotData)
	}

	return result
}

// replaceLink 链接转换
// 输入: https://www.ithome.com/0/741/963.htm
// 输出ID: 741963
// 输出移动链接: https://m.ithome.com/html/741963.htm
func (h *IthomeXijiayiHandler) replaceLink(url string) (string, string) {
	pattern := regexp.MustCompile(`https://www\.ithome\.com/0/(\d+)/(\d+)\.htm`)
	matches := pattern.FindStringSubmatch(url)

	if len(matches) == 3 {
		id := matches[1] + matches[2]
		mobileURL := fmt.Sprintf("https://m.ithome.com/html/%s.htm", id)
		return id, mobileURL
	}

	return "100000", url
}

// parseTime 解析时间字符串为时间戳
// 输入格式：2024-01-15 12:30:00
func (h *IthomeXijiayiHandler) parseTime(timeStr string) string {
	// 尝试解析时间
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return ""
	}

	return strconv.FormatInt(t.Unix(), 10)
}
