package routes

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

// ProductHuntHandler Product Hunt处理器
type ProductHuntHandler struct {
	fetcher *service.Fetcher
}

// NewProductHuntHandler 创建Product Hunt处理器
func NewProductHuntHandler(fetcher *service.Fetcher) *ProductHuntHandler {
	return &ProductHuntHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *ProductHuntHandler) GetPath() string {
	return "/producthunt"
}

// Handle 处理请求
func (h *ProductHuntHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchProductHuntHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"producthunt",  // name: 平台调用名称
		"Product Hunt", // title: 平台显示名称
		"Today",        // type: 榜单类型
		"发现每日热门产品与创新应用",                // description: 平台描述
		"https://www.producthunt.com/", // link: 官方链接
		nil,                            // params: 无参数映射
		data,                           // data: 热榜数据
		!noCache,                       // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchProductHuntHot 从Product Hunt获取数据
func (h *ProductHuntHandler) fetchProductHuntHot(ctx context.Context) ([]models.HotData, error) {
	feedURL := "https://www.producthunt.com/feed"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(feedURL, map[string]string{
		"Accept":          "application/atom+xml, application/xml;q=0.9, */*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
	})
	if err != nil {
		return nil, fmt.Errorf("请求Product Hunt失败: %w", err)
	}

	// 解析 feed
	data, err := h.parseFeed(body)
	if err != nil {
		return nil, fmt.Errorf("解析 Product Hunt feed 失败: %w", err)
	}
	return data, nil
}

// parseFeed 解析 Product Hunt Atom feed
func (h *ProductHuntHandler) parseFeed(data []byte) ([]models.HotData, error) {
	result := make([]models.HotData, 0)

	var feed productHuntFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return result, err
	}

	for _, entry := range feed.Entries {
		title := strings.TrimSpace(entry.Title)
		if title == "" {
			continue
		}

		url := entry.FirstLink()
		if url == "" {
			continue
		}

		id := extractProductHuntID(entry.ID)
		timestamp := timeutil.ParseTime(entry.Updated)
		if timestamp == 0 {
			timestamp = timeutil.ParseTime(entry.Published)
		}

		hotData := models.HotData{
			ID:        id,
			Title:     title,
			URL:       url,
			MobileURL: url,
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result, nil
}

type productHuntFeed struct {
	Entries []productHuntEntry `xml:"entry"`
}

type productHuntEntry struct {
	ID        string            `xml:"id"`
	Title     string            `xml:"title"`
	Updated   string            `xml:"updated"`
	Published string            `xml:"published"`
	Links     []productHuntLink `xml:"link"`
}

func (e productHuntEntry) FirstLink() string {
	for _, link := range e.Links {
		if link.Rel == "" || link.Rel == "alternate" {
			return link.Href
		}
	}
	return ""
}

type productHuntLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func extractProductHuntID(raw string) string {
	// 原始 ID 格式如: tag:www.producthunt.com,2005:Post/1028201
	if raw == "" {
		return ""
	}
	if idx := strings.LastIndex(raw, "/"); idx != -1 && idx+1 < len(raw) {
		return raw[idx+1:]
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		return parts[len(parts)-1]
	}
	return raw
}
