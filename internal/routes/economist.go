package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

const economistFeedURL = "https://feed2json.org/convert?url=https://www.economist.com/latest/rss.xml"

// EconomistHandler The Economist 最新文章处理器
type EconomistHandler struct {
	fetcher *service.Fetcher
}

// NewEconomistHandler 创建处理器
func NewEconomistHandler(fetcher *service.Fetcher) *EconomistHandler {
	return &EconomistHandler{fetcher: fetcher}
}

// GetPath 返回路由路径
func (h *EconomistHandler) GetPath() string {
	return "/economist"
}

// Handle 入口
func (h *EconomistHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	data, err := h.fetchEconomist(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	resp := models.SuccessResponse(
		"economist",
		"The Economist",
		"Latest",
		"The Economist 最新深度报道精选",
		"https://www.economist.com/latest",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

func (h *EconomistHandler) fetchEconomist(ctx context.Context) ([]models.HotData, error) {
	httpClient := h.fetcher.GetHTTPClient()

	headers := map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.9",
	}

	body, err := httpClient.Get(economistFeedURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 The Economist feed 失败: %w", err)
	}

	var feed economistFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 The Economist feed 失败: %w", err)
	}

	return transformEconomistItems(feed.Items), nil
}

func transformEconomistItems(items []economistFeedItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for idx, item := range items {
		title := strings.TrimSpace(item.Title)
		link := strings.TrimSpace(item.URL)
		if title == "" || link == "" {
			continue
		}

		desc := strings.TrimSpace(item.ContentText)
		if desc == "" {
			desc = strings.TrimSpace(item.Summary)
		}

		id := item.GUID
		if id == "" {
			id = fmt.Sprintf("%d", idx)
		}

		timestamp := timeutil.ParseTime(item.DatePublished)

		result = append(result, models.HotData{
			ID:        id,
			Title:     title,
			Desc:      desc,
			Author:    strings.TrimSpace(item.Author.Name),
			Timestamp: timestamp,
			URL:       link,
			MobileURL: link,
		})
	}

	return result
}

type economistFeedResponse struct {
	Items []economistFeedItem `json:"items"`
}

type economistFeedItem struct {
	GUID          string                `json:"guid"`
	URL           string                `json:"url"`
	Title         string                `json:"title"`
	Summary       string                `json:"summary"`
	ContentText   string                `json:"content_text"`
	ContentHTML   string                `json:"content_html"`
	DatePublished string                `json:"date_published"`
	Author        economistFeedAuthor   `json:"author"`
	Attachments   []economistAttachment `json:"attachments"`
}

type economistFeedAuthor struct {
	Name string `json:"name"`
}

type economistAttachment struct {
	URL string `json:"url"`
}
