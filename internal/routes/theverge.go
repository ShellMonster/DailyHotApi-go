package routes

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

const theVergeFeedURL = "https://www.theverge.com/rss/index.xml"

// TheVergeHandler The Verge 科技资讯处理器
type TheVergeHandler struct {
	fetcher *service.Fetcher
}

// NewTheVergeHandler 创建 The Verge 处理器
func NewTheVergeHandler(fetcher *service.Fetcher) *TheVergeHandler {
	return &TheVergeHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *TheVergeHandler) GetPath() string {
	return "/theverge"
}

// Handle 处理请求
func (h *TheVergeHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	data, err := h.fetchTheVerge(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	resp := models.SuccessResponse(
		"theverge",
		"The Verge",
		"Latest",
		"关注 The Verge 最新科技与文化报道",
		"https://www.theverge.com/",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchTheVerge 拉取并转换 The Verge Atom feed
func (h *TheVergeHandler) fetchTheVerge(ctx context.Context) ([]models.HotData, error) {
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(theVergeFeedURL, map[string]string{
		"Accept":          "application/atom+xml, application/xml;q=0.9, */*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
	})
	if err != nil {
		return nil, fmt.Errorf("请求 The Verge RSS 失败: %w", err)
	}

	var feed vergeFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 The Verge RSS 失败: %w", err)
	}

	return h.transformData(feed.Entries), nil
}

func (h *TheVergeHandler) transformData(items []vergeEntry) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		link := strings.TrimSpace(item.FirstLink())
		if title == "" || link == "" {
			continue
		}

		desc := strings.TrimSpace(item.Summary)
		if desc == "" {
			desc = strings.TrimSpace(item.Content)
		}

		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = link
		}

		timestamp := parseAtomTime(item.Updated)
		if timestamp == 0 {
			timestamp = parseAtomTime(item.Published)
		}

		hotData := models.HotData{
			ID:        id,
			Title:     title,
			Desc:      desc,
			Author:    strings.TrimSpace(item.Author.Name),
			Timestamp: timestamp,
			URL:       link,
			MobileURL: link,
		}

		result = append(result, hotData)
	}

	return result
}

// parseAtomTime 解析 Atom feed 中的时间为毫秒级时间戳
func parseAtomTime(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UnixMilli()
	}

	return 0
}

type vergeFeed struct {
	Entries []vergeEntry `xml:"entry"`
}

type vergeEntry struct {
	ID         string          `xml:"id"`
	Title      string          `xml:"title"`
	Updated    string          `xml:"updated"`
	Published  string          `xml:"published"`
	Summary    string          `xml:"summary"`
	Content    string          `xml:"content"`
	Links      []vergeLink     `xml:"link"`
	Author     vergeAuthor     `xml:"author"`
	Categories []vergeCategory `xml:"category"`
}

func (e vergeEntry) FirstLink() string {
	for _, link := range e.Links {
		if link.Rel == "" || link.Rel == "alternate" {
			return link.Href
		}
	}
	return ""
}

type vergeLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type vergeAuthor struct {
	Name string `xml:"name"`
}

type vergeCategory struct {
	Term string `xml:"term,attr"`
}
