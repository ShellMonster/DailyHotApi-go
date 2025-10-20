package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// ZhihuDailyHandler 知乎日报处理器
type ZhihuDailyHandler struct {
	fetcher *service.Fetcher
}

// NewZhihuDailyHandler 创建知乎日报处理器
func NewZhihuDailyHandler(fetcher *service.Fetcher) *ZhihuDailyHandler {
	return &ZhihuDailyHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *ZhihuDailyHandler) GetPath() string {
	return "/zhihu-daily"
}

// Handle 处理请求
func (h *ZhihuDailyHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchZhihuDaily(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		"zhihu_daily",
		"知乎日报",
		"推荐榜",
		"知乎日报推荐榜",
		"https://daily.zhihu.com",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchZhihuDaily 从知乎日报 API 获取数据
func (h *ZhihuDailyHandler) fetchZhihuDaily(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://daily.zhihu.com/api/4/news/latest"

	// 发起 HTTP 请求(需要特定 headers)
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"Referer": "https://daily.zhihu.com/api/4/news/latest",
		"Host":    "daily.zhihu.com",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求知乎日报 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp ZhihuDailyAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析知乎日报响应失败: %w", err)
	}

	// 过滤掉 type != 0 的项目
	filteredStories := make([]ZhihuDailyStory, 0)
	for _, story := range apiResp.Stories {
		if story.Type == 0 {
			filteredStories = append(filteredStories, story)
		}
	}

	// 转换为统一格式
	return h.transformData(filteredStories), nil
}

// transformData 将知乎日报原始数据转换为统一格式
func (h *ZhihuDailyHandler) transformData(items []ZhihuDailyStory) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 获取封面图(第一张)
		cover := ""
		if len(item.Images) > 0 {
			cover = item.Images[0]
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     item.Title,
			Cover:     cover,
			Author:    item.Hint,
			URL:       item.URL,
			MobileURL: item.URL,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是知乎日报 API 的响应结构体定义

// ZhihuDailyAPIResponse 知乎日报 API 响应
type ZhihuDailyAPIResponse struct {
	Stories []ZhihuDailyStory `json:"stories"`
}

// ZhihuDailyStory 单个故事项
type ZhihuDailyStory struct {
	ID     int64    `json:"id"`     // 故事 ID
	Title  string   `json:"title"`  // 标题
	Images []string `json:"images"` // 图片列表
	Hint   string   `json:"hint"`   // 提示/作者
	URL    string   `json:"url"`    // 链接
	Type   int      `json:"type"`   // 类型(0 为正常文章)
}
