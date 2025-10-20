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

// HonkaiHandler 崩坏3处理器
type HonkaiHandler struct {
	fetcher *service.Fetcher
}

// NewHonkaiHandler 创建崩坏3处理器
func NewHonkaiHandler(fetcher *service.Fetcher) *HonkaiHandler {
	return &HonkaiHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HonkaiHandler) GetPath() string {
	return "/honkai"
}

// Handle 处理请求
func (h *HonkaiHandler) Handle(c *fiber.Ctx) error {
	newsType := c.Query("type", "1") // 默认公告
	noCache := c.Query("cache") == "false"

	data, err := h.fetchHonkai(c.Context(), newsType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		fmt.Sprintf("honkai_%s", newsType),
		"崩坏3",
		"最新动态",
		"崩坏3最新动态列表",
		"https://www.miyoushe.com/bh3",
		nil,
		data,
		!noCache,
	))
}

// fetchHonkai 从米游社 API 获取崩坏3数据
func (h *HonkaiHandler) fetchHonkai(ctx context.Context, newsType string) ([]models.HotData, error) {
	// gids=1 是崩坏3
	apiURL := fmt.Sprintf("https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?client_type=4&gids=1&last_id=&page_size=20&type=%s", newsType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求米游社 API 失败: %w", err)
	}

	// 解析 JSON 响应(使用 genshin.go 中定义的结构体)
	var apiResp MiyousheAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析米游社响应失败: %w", err)
	}

	// 转换为统一格式(使用 bh3 作为游戏标识)
	return h.transformData(apiResp.Data.List), nil
}

// transformData 将米游社原始数据转换为统一格式
func (h *HonkaiHandler) transformData(items []MiyousheListItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		post := item.Post

		// 获取封面图
		cover := post.Cover
		if cover == "" && len(post.Images) > 0 {
			cover = post.Images[0]
		}

		// 作者昵称
		author := ""
		if item.User != nil {
			author = item.User.Nickname
		}

		// 时间戳转换
		timestamp := strconv.FormatInt(post.CreatedAt, 10)

		hotData := models.HotData{
			ID:        post.PostID,
			Title:     post.Subject,
			Desc:      post.Content,
			Cover:     cover,
			Author:    author,
			Hot:       post.ViewStatus,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.miyoushe.com/bh3/article/%s", post.PostID),
			MobileURL: fmt.Sprintf("https://m.miyoushe.com/bh3/#/article/%s", post.PostID),
		}

		result = append(result, hotData)
	}

	return result
}
