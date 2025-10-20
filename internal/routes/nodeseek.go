package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// NodeseekHandler NodeSeek 处理器
type NodeseekHandler struct {
	fetcher *service.Fetcher
}

// NewNodeseekHandler 创建 NodeSeek 处理器
func NewNodeseekHandler(fetcher *service.Fetcher) *NodeseekHandler {
	return &NodeseekHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *NodeseekHandler) GetPath() string {
	return "/nodeseek"
}

// Handle 处理请求
func (h *NodeseekHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchNodeseek(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		"nodeseek_latest",
		"NodeSeek",
		"最新",
		"NodeSeek 最新帖子",
		"https://www.nodeseek.com",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchNodeseek 从 NodeSeek RSS 获取数据
func (h *NodeseekHandler) fetchNodeseek(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://feed2json.org/convert?url=https://rss.nodeseek.com/"

	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":     "application/json",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 NodeSeek feed 失败: %w", err)
	}

	var feed nodeseekFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 NodeSeek feed 失败: %w", err)
	}

	return h.transformData(feed.Items), nil
}

// transformData 将 RSS 数据转换为统一格式
func (h *NodeseekHandler) transformData(items []nodeseekFeedItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for i, item := range items {
		// 获取 ID(使用 GUID 或索引)
		id := item.GUID
		if id == "" {
			id = fmt.Sprintf("%d", i)
		}

		// 描述(优先使用 Content,然后 Description)
		// TypeScript 版本优先使用 content 字段
		desc := item.ContentHTML
		if desc == "" {
			desc = item.ContentText
		}
		// 去掉前后空白
		desc = strings.TrimSpace(desc)

		// 时间戳转换为毫秒级整数（与 TypeScript 版本保持一致）
		var timestamp int64
		if item.DatePublished != "" {
			timestamp = parseTimeToMillis(item.DatePublished)
		}

		// 作者
		author := strings.TrimSpace(item.Author.Name)

		hotData := models.HotData{
			ID:        id,
			Title:     item.Title,
			Desc:      desc,
			Author:    author,
			Timestamp: timestamp,
			URL:       item.URL,
			MobileURL: item.URL,
		}

		result = append(result, hotData)
	}

	return result
}

type nodeseekFeedResponse struct {
	Items []nodeseekFeedItem `json:"items"`
}

type nodeseekFeedItem struct {
	GUID          string             `json:"guid"`
	URL           string             `json:"url"`
	Title         string             `json:"title"`
	ContentHTML   string             `json:"content_html"`
	ContentText   string             `json:"content_text"`
	DatePublished string             `json:"date_published"`
	Author        nodeseekFeedAuthor `json:"author"`
}

type nodeseekFeedAuthor struct {
	Name string `json:"name"`
}

func parseTimeToMillis(value string) int64 {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}
