package routes

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

const techCrunchFeedURL = "https://techcrunch.com/feed/"

// TechCrunchHandler TechCrunch 科技资讯处理器
type TechCrunchHandler struct {
	fetcher *service.Fetcher
}

// NewTechCrunchHandler 创建 TechCrunch 处理器
func NewTechCrunchHandler(fetcher *service.Fetcher) *TechCrunchHandler {
	return &TechCrunchHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *TechCrunchHandler) GetPath() string {
	return "/techcrunch"
}

// Handle 处理请求
func (h *TechCrunchHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	data, err := h.fetchTechCrunch(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	resp := models.SuccessResponse(
		"techcrunch",
		"TechCrunch",
		"Top Stories",
		"追踪全球最新的科技创业资讯",
		"https://techcrunch.com/",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchTechCrunch 拉取并转换 TechCrunch RSS 数据
func (h *TechCrunchHandler) fetchTechCrunch(ctx context.Context) ([]models.HotData, error) {
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(techCrunchFeedURL, map[string]string{
		"Accept":          "application/rss+xml, application/xml;q=0.9, */*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
	})
	if err != nil {
		return nil, fmt.Errorf("请求 TechCrunch RSS 失败: %w", err)
	}

	var feed techCrunchRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 TechCrunch RSS 失败: %w", err)
	}

	return h.transformData(feed.Channel.Items), nil
}

func (h *TechCrunchHandler) transformData(items []techCrunchItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		link := strings.TrimSpace(item.Link)
		if title == "" || link == "" {
			continue
		}

		desc := strings.TrimSpace(item.Description)
		if desc == "" {
			desc = strings.TrimSpace(item.Content)
		}

		id := strings.TrimSpace(item.GUID)
		if id == "" {
			id = link
		}

		timestamp := parsePubDate(item.PubDate)

		hotData := models.HotData{
			ID:        id,
			Title:     title,
			Desc:      desc,
			Author:    strings.TrimSpace(item.Creator),
			Timestamp: timestamp,
			URL:       link,
			MobileURL: link,
		}

		result = append(result, hotData)
	}

	return result
}

// parsePubDate 解析 RSS pubDate 字段为毫秒级时间戳
func parsePubDate(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
	}

	for _, layout := range layouts {
		if ts, err := time.Parse(layout, value); err == nil {
			return ts.UnixMilli()
		}
	}

	return timeutil.ParseTime(value)
}

type techCrunchRSS struct {
	Channel techCrunchChannel `xml:"channel"`
}

type techCrunchChannel struct {
	Items []techCrunchItem `xml:"item"`
}

type techCrunchItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Creator     string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
}
