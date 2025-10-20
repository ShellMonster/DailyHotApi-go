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

// HelloGitHubHandler HelloGitHub处理器
type HelloGitHubHandler struct {
	fetcher *service.Fetcher
}

// NewHelloGitHubHandler 创建HelloGitHub处理器
func NewHelloGitHubHandler(fetcher *service.Fetcher) *HelloGitHubHandler {
	return &HelloGitHubHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HelloGitHubHandler) GetPath() string {
	return "/hellogithub"
}

// Handle 处理请求
func (h *HelloGitHubHandler) Handle(c *fiber.Ctx) error {
	// 支持排序: featured-精选, all-全部
	sortType := c.Query("sort", "featured")
	noCache := c.Query("cache") == "false"

	data, err := h.fetchHelloGitHubHot(c.Context(), sortType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		fmt.Sprintf("hellogithub_%s", sortType),
		"HelloGitHub",
		"热门仓库",
		"HelloGitHub热门仓库列表",
		"https://hellogithub.com",
		nil,
		data,
		!noCache,
	))
}

// fetchHelloGitHubHot 从HelloGitHub API 获取数据
func (h *HelloGitHubHandler) fetchHelloGitHubHot(ctx context.Context, sortType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://abroad.hellogithub.com/v1/?sort_by=%s&tid=&page=1", sortType)

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求HelloGitHub API 失败: %w", err)
	}

	var apiResp HelloGitHubAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析HelloGitHub响应失败: %w", err)
	}

	return h.transformData(apiResp.Data), nil
}

// transformData 转换数据格式
func (h *HelloGitHubHandler) transformData(items []HelloGitHubItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理UpdatedAt字段(可能是int64或string)
		var timestamp string
		switch v := item.UpdatedAt.(type) {
		case float64:
			timestamp = strconv.FormatInt(int64(v), 10)
		case int64:
			timestamp = strconv.FormatInt(v, 10)
		case string:
			timestamp = v
		default:
			timestamp = ""
		}

		hotData := models.HotData{
			ID:        item.ItemID,
			Title:     item.Title,
			Desc:      item.Summary,
			Author:    item.Author,
			Hot:       item.ClicksTotal,
			URL:       fmt.Sprintf("https://hellogithub.com/repository/%s", item.ItemID),
			MobileURL: fmt.Sprintf("https://hellogithub.com/repository/%s", item.ItemID),
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result
}

// HelloGitHubAPIResponse HelloGitHub API 响应
type HelloGitHubAPIResponse struct {
	Data []HelloGitHubItem `json:"data"`
}

// HelloGitHubItem 仓库项
type HelloGitHubItem struct {
	ItemID      string      `json:"item_id"`
	Title       string      `json:"title"`
	Summary     string      `json:"summary"`
	Author      string      `json:"author"`
	ClicksTotal int64       `json:"clicks_total"`
	UpdatedAt   interface{} `json:"updated_at"` // 可能是int64或string
}
