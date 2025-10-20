package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// HackerNewsHandler Hacker News 处理器
type HackerNewsHandler struct {
	fetcher *service.Fetcher
}

// NewHackerNewsHandler 创建 Hacker News 处理器
func NewHackerNewsHandler(fetcher *service.Fetcher) *HackerNewsHandler {
	return &HackerNewsHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HackerNewsHandler) GetPath() string {
	return "/hackernews"
}

// Handle 处理请求
func (h *HackerNewsHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchHackerNews(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"hackernews",                    // name: 平台调用名称
		"Hacker News",                   // title: 平台显示名称
		"Popular",                       // type: 榜单类型
		"发现编程与科技的最新动态",                  // description: 平台描述
		"https://news.ycombinator.com/", // link: 官方链接
		nil,                             // params: 无参数映射
		data,                            // data: 热榜数据
		!noCache,                        // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchHackerNews 从 Hacker News 获取数据
func (h *HackerNewsHandler) fetchHackerNews(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://hn.algolia.com/api/v1/search?tags=front_page"

	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":     "application/json",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 Hacker News 失败: %w", err)
	}

	var apiResp hackerNewsAlgoliaResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 Hacker News 响应失败: %w", err)
	}

	result := make([]models.HotData, 0, len(apiResp.Hits))
	for _, hit := range apiResp.Hits {
		title := hit.Title
		if title == "" {
			title = hit.StoryTitle
		}
		if title == "" {
			continue
		}

		url := hit.URL
		if url == "" {
			url = hit.StoryURL
		}
		if url == "" {
			url = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}

		hotData := models.HotData{
			ID:        hit.ObjectID,
			Title:     strings.TrimSpace(title),
			Author:    hit.Author,
			Hot:       int64(hit.Points),
			URL:       url,
			MobileURL: url,
		}

		result = append(result, hotData)
	}

	return result, nil
}

type hackerNewsAlgoliaResponse struct {
	Hits []hackerNewsHit `json:"hits"`
}

type hackerNewsHit struct {
	ObjectID   string `json:"objectID"`
	Title      string `json:"title"`
	StoryTitle string `json:"story_title"`
	URL        string `json:"url"`
	StoryURL   string `json:"story_url"`
	Author     string `json:"author"`
	Points     int    `json:"points"`
}
