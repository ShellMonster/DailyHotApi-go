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

// DoubanGroupHandler 豆瓣讨论处理器
type DoubanGroupHandler struct {
	fetcher *service.Fetcher
}

// NewDoubanGroupHandler 创建豆瓣讨论处理器
func NewDoubanGroupHandler(fetcher *service.Fetcher) *DoubanGroupHandler {
	return &DoubanGroupHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *DoubanGroupHandler) GetPath() string {
	return "/douban-group"
}

// Handle 处理请求
func (h *DoubanGroupHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchDoubanGroup(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"douban_group",
		"豆瓣讨论",
		"讨论精选",
		"豆瓣讨论精选列表",
		"https://www.douban.com/group/explore",
		nil,
		data,
		!noCache,
	))
}

// fetchDoubanGroup 从豆瓣获取讨论数据
func (h *DoubanGroupHandler) fetchDoubanGroup(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.douban.com/group/explore"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求豆瓣失败: %w", err)
	}

	// 解析 HTML
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析 HTML 提取讨论列表
func (h *DoubanGroupHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return result
	}

	doc.Find(".article .channel-item").Each(func(i int, s *goquery.Selection) {
		link := s.Find("h3 a")
		url, ok := link.Attr("href")
		if !ok || url == "" {
			return
		}

		title := strings.TrimSpace(link.Text())
		if title == "" {
			return
		}

		cover, _ := s.Find(".pic-wrap img").Attr("src")
		desc := strings.TrimSpace(s.Find(".block p").Text())
		pubtime := strings.TrimSpace(s.Find("span.pubtime").Text())

		id := getNumbersFromURL(url)
		timestamp := timeutil.ParseTime(pubtime)

		hotData := models.HotData{
			ID:        strconv.FormatInt(id, 10),
			Title:     title,
			Desc:      desc,
			Cover:     cover,
			Timestamp: timestamp,
			URL:       url,
			MobileURL: fmt.Sprintf("https://m.douban.com/group/topic/%d/", id),
		}

		result = append(result, hotData)
	})

	return result
}

// extractID 从 URL 中提取 ID
func (h *DoubanGroupHandler) extractID(url string) string {
	return strconv.FormatInt(getNumbersFromURL(url), 10)
}

func getNumbersFromURL(url string) int64 {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(url)
	if match == "" {
		return 100000000
	}
	if num, err := strconv.ParseInt(match, 10, 64); err == nil {
		return num
	}
	return 100000000
}
