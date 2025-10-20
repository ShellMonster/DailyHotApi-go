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

const guardianFeedURL = "https://feed2json.org/convert?url=https://www.theguardian.com/world/rss"

// GuardianHandler The Guardian 世界新闻处理器
type GuardianHandler struct {
	fetcher *service.Fetcher
}

// NewGuardianHandler 创建处理器
func NewGuardianHandler(fetcher *service.Fetcher) *GuardianHandler {
	return &GuardianHandler{fetcher: fetcher}
}

// GetPath 返回路由路径
func (h *GuardianHandler) GetPath() string {
	return "/theguardian"
}

// Handle 入口
func (h *GuardianHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	data, err := h.fetchGuardian(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	resp := models.SuccessResponse(
		"theguardian",
		"The Guardian",
		"World News",
		"英媒 The Guardian 世界新闻精选",
		"https://www.theguardian.com/world",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

func (h *GuardianHandler) fetchGuardian(ctx context.Context) ([]models.HotData, error) {
	httpClient := h.fetcher.GetHTTPClient()

	headers := map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.9",
	}

	body, err := httpClient.Get(guardianFeedURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 The Guardian feed 失败: %w", err)
	}

	var feed guardianFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 The Guardian feed 失败: %w", err)
	}

	return transformGuardianItems(feed.Items), nil
}

func transformGuardianItems(items []guardianFeedItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for idx, item := range items {
		title := strings.TrimSpace(item.Title)
		link := strings.TrimSpace(item.URL)
		if title == "" || link == "" {
			continue
		}

		desc := item.ContentText
		if desc == "" {
			desc = item.Summary
		}
		desc = strings.TrimSpace(desc)

		id := item.GUID
		if id == "" {
			id = fmt.Sprintf("%d", idx)
		}

		timestamp := timeutil.ParseTime(item.DatePublished)
		author := strings.TrimSpace(item.Author.Name)

		result = append(result, models.HotData{
			ID:        id,
			Title:     title,
			Desc:      desc,
			Author:    author,
			Timestamp: timestamp,
			URL:       link,
			MobileURL: link,
		})
	}

	return result
}

type guardianFeedResponse struct {
	Items []guardianFeedItem `json:"items"`
}

type guardianFeedItem struct {
	GUID          string               `json:"guid"`
	URL           string               `json:"url"`
	Title         string               `json:"title"`
	Summary       string               `json:"summary"`
	ContentText   string               `json:"content_text"`
	ContentHTML   string               `json:"content_html"`
	DatePublished string               `json:"date_published"`
	Author        guardianFeedAuthor   `json:"author"`
	Tags          []guardianFeedTag    `json:"tags"`
	Attachments   []guardianAttachment `json:"attachments"`
}

type guardianFeedAuthor struct {
	Name string `json:"name"`
}

type guardianFeedTag struct {
	Name string `json:"name"`
}

type guardianAttachment struct {
	URL string `json:"url"`
}
