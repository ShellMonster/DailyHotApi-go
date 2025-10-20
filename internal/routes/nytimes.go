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

// NYTimesHandler 纽约时报处理器
type NYTimesHandler struct {
	fetcher *service.Fetcher
}

// NewNYTimesHandler 创建纽约时报处理器
func NewNYTimesHandler(fetcher *service.Fetcher) *NYTimesHandler {
	return &NYTimesHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *NYTimesHandler) GetPath() string {
	return "/nytimes"
}

// Handle 处理请求
func (h *NYTimesHandler) Handle(c *fiber.Ctx) error {
	areaType := c.Query("type", "china") // 默认中文网
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchNYTimes(c.Context(), areaType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		fmt.Sprintf("nytimes_%s", areaType),
		"纽约时报",
		h.getAreaName(areaType),
		"纽约时报新闻",
		"https://www.nytimes.com",
		map[string]interface{}{"type": areaType},
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// getAreaName 获取地区名称
func (h *NYTimesHandler) getAreaName(areaType string) string {
	areaMap := map[string]string{
		"china":  "中文网",
		"global": "全球版",
	}
	if name, ok := areaMap[areaType]; ok {
		return name
	}
	return "中文网"
}

// fetchNYTimes 从纽约时报 RSS 获取数据
func (h *NYTimesHandler) fetchNYTimes(ctx context.Context, areaType string) ([]models.HotData, error) {
	// 选择 RSS 地址
	rssURL := "https://feed2json.org/convert?url=https://cn.nytimes.com/rss/"
	if areaType == "global" {
		rssURL = "https://feed2json.org/convert?url=https://rss.nytimes.com/services/xml/rss/nyt/World.xml"
	}

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":     "application/json",
	}

	body, err := httpClient.Get(rssURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求纽约时报 RSS 失败: %w", err)
	}

	var feed nyTimesFeedResponse
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析 RSS 失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(feed.Items), nil
}

// transformData 将 RSS 数据转换为统一格式
func (h *NYTimesHandler) transformData(items []nyTimesFeedItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for i, item := range items {
		// 获取 ID(使用 GUID 或索引)
		id := item.GUID
		if id == "" {
			id = fmt.Sprintf("%d", i)
		}

		// 描述(优先使用 Description,否则 Content)
		desc := strings.TrimSpace(item.ContentHTML)
		if desc == "" && item.ContentText != "" {
			desc = strings.TrimSpace(item.ContentText)
		}

		// 时间戳
		timestamp := ""
		if item.DatePublished != "" {
			if t := parseNYTTime(item.DatePublished); !t.IsZero() {
				timestamp = t.Format(time.RFC3339)
			}
		}

		// 作者
		author := ""
		if item.Author.Name != "" {
			author = item.Author.Name
		}

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

type nyTimesFeedResponse struct {
	Items []nyTimesFeedItem `json:"items"`
}

type nyTimesFeedItem struct {
	GUID          string            `json:"guid"`
	URL           string            `json:"url"`
	Title         string            `json:"title"`
	ContentHTML   string            `json:"content_html"`
	ContentText   string            `json:"content_text"`
	DatePublished string            `json:"date_published"`
	Author        nyTimesFeedAuthor `json:"author"`
}

type nyTimesFeedAuthor struct {
	Name string `json:"name"`
}

func parseNYTTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return t
	}
	// feed2json 可能返回 RFC1123 样式
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
	}
	for _, layout := range layouts {
		if tt, err := time.Parse(layout, value); err == nil {
			return tt
		}
	}
	return time.Time{}
}
