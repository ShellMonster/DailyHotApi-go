package routes

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

// GameresHandler GameRes 游资网处理器
type GameresHandler struct {
	fetcher *service.Fetcher
}

// NewGameresHandler 创建 GameRes 处理器
func NewGameresHandler(fetcher *service.Fetcher) *GameresHandler {
	return &GameresHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *GameresHandler) GetPath() string {
	return "/gameres"
}

// Handle 处理请求
func (h *GameresHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchGameres(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"gameres_news",
		"GameRes 游资网",
		"最新资讯",
		"GameRes 游资网最新资讯列表",
		"https://www.gameres.com",
		nil,
		data,
		!noCache,
	))
}

// fetchGameres 从 GameRes 网站获取数据
func (h *GameresHandler) fetchGameres(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.gameres.com"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求 GameRes 失败: %w", err)
	}

	// 解析 HTML
	return h.parseHTML(string(body)), nil
}

// parseHTML 解析 HTML 提取新闻列表
func (h *GameresHandler) parseHTML(html string) []models.HotData {
	result := make([]models.HotData, 0)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return result
	}

	doc.Find(`div[data-news-pane-id="100000"] article.feed-item`).Each(func(i int, s *goquery.Selection) {
		titleSelection := s.Find(".feed-item-title-a").First()
		title := strings.TrimSpace(titleSelection.Text())
		if title == "" {
			return
		}

		href, ok := titleSelection.Attr("href")
		if !ok || href == "" {
			return
		}

		url := href
		if strings.HasPrefix(url, "/") {
			url = fmt.Sprintf("https://www.gameres.com%s", url)
		}

		cover, _ := s.Find(".thumb").Attr("data-original")
		if cover == "" {
			cover, _ = s.Find(".thumb").Attr("src")
		}

		desc := strings.TrimSpace(s.Find(".feed-item-right > p").First().Text())
		markInfo := strings.TrimSpace(s.Find(".mark-info").Contents().First().Text())
		timestamp := timeutil.ParseTime(markInfo)

		hotData := models.HotData{
			ID:        url,
			Title:     title,
			Desc:      desc,
			Cover:     cover,
			Timestamp: timestamp,
			URL:       url,
			MobileURL: url,
		}

		result = append(result, hotData)

		// 限制数量
		if len(result) >= 30 {
			return
		}
	})

	return result
}
