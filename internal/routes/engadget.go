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

const engadgetFeedURL = "https://feed2json.org/convert?url=https://www.engadget.com/rss.xml"

// EngadgetHandler Engadget 科技快讯处理器
type EngadgetHandler struct {
	fetcher *service.Fetcher
}

// NewEngadgetHandler 创建处理器
func NewEngadgetHandler(fetcher *service.Fetcher) *EngadgetHandler {
	return &EngadgetHandler{fetcher: fetcher}
}

// GetPath 返回路由路径
func (h *EngadgetHandler) GetPath() string {
	return "/engadget"
}

// Handle 入口
func (h *EngadgetHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	data, err := h.fetchEngadget(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	resp := models.SuccessResponse(
		"engadget",
		"Engadget",
		"Top Stories",
		"Engadget 每日最新科技与数码资讯",
		"https://www.engadget.com/",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

func (h *EngadgetHandler) fetchEngadget(ctx context.Context) ([]models.HotData, error) {
	httpClient := h.fetcher.GetHTTPClient()

	headers := map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.9",
	}

	body, err := httpClient.Get(engadgetFeedURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 Engadget feed 失败: %w", err)
	}

	var feed engadgetFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 Engadget feed 失败: %w", err)
	}

	return transformEngadgetItems(feed.Items), nil
}

func transformEngadgetItems(items []engadgetFeedItem) []models.HotData {
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

		author := strings.TrimSpace(item.Author.Name)
		timestamp := timeutil.ParseTime(item.DatePublished)

		result = append(result, models.HotData{
			ID:        id,
			Title:     title,
			Desc:      desc,
			Author:    author,
			Timestamp: timestamp,
			URL:       link,
			MobileURL: link,
			Cover:     extractFirstAttachment(item.Attachments),
		})
	}

	return result
}

func extractFirstAttachment(attachments []engadgetAttachment) string {
	for _, att := range attachments {
		if att.URL != "" {
			return att.URL
		}
	}
	return ""
}

type engadgetFeedResponse struct {
	Items []engadgetFeedItem `json:"items"`
}

type engadgetFeedItem struct {
	GUID          string               `json:"guid"`
	URL           string               `json:"url"`
	Title         string               `json:"title"`
	Summary       string               `json:"summary"`
	ContentText   string               `json:"content_text"`
	ContentHTML   string               `json:"content_html"`
	DatePublished string               `json:"date_published"`
	Author        engadgetFeedAuthor   `json:"author"`
	Attachments   []engadgetAttachment `json:"attachments"`
}

type engadgetFeedAuthor struct {
	Name string `json:"name"`
}

type engadgetAttachment struct {
	URL string `json:"url"`
}
