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

// StarrailHandler 崩坏:星穹铁道处理器
type StarrailHandler struct {
	fetcher *service.Fetcher
}

// NewStarrailHandler 创建崩坏:星穹铁道处理器
func NewStarrailHandler(fetcher *service.Fetcher) *StarrailHandler {
	return &StarrailHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *StarrailHandler) GetPath() string {
	return "/starrail"
}

// Handle 处理请求
func (h *StarrailHandler) Handle(c *fiber.Ctx) error {
	newsType := c.Query("type", "1") // 默认公告
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchStarrail(c.Context(), newsType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		fmt.Sprintf("starrail_%s", newsType),
		"崩坏：星穹铁道",
		"最新动态",
		"崩坏：星穹铁道最新动态",
		"https://www.miyoushe.com/sr/",
		map[string]interface{}{"type": newsType},
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchStarrail 从米游社 API 获取星穹铁道数据
func (h *StarrailHandler) fetchStarrail(ctx context.Context, newsType string) ([]models.HotData, error) {
	// gids=6 是崩坏:星穹铁道
	apiURL := fmt.Sprintf("https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?client_type=4&gids=6&page_size=20&type=%s", newsType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求米游社 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp MiyousheAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析米游社响应失败: %w", err)
	}

	// 转换为统一格式(使用 sr 作为游戏标识)
	return h.transformData(apiResp.Data.List), nil
}

// transformData 将米游社原始数据转换为统一格式
func (h *StarrailHandler) transformData(items []MiyousheListItem) []models.HotData {
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
			URL:       fmt.Sprintf("https://www.miyoushe.com/sr/article/%s", post.PostID),
			MobileURL: fmt.Sprintf("https://m.miyoushe.com/sr/#/article/%s", post.PostID),
		}

		result = append(result, hotData)
	}

	return result
}
