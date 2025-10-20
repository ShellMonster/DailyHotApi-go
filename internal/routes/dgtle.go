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

// DgtleHandler 数字尾巴处理器
type DgtleHandler struct {
	fetcher *service.Fetcher
}

// NewDgtleHandler 创建数字尾巴处理器
func NewDgtleHandler(fetcher *service.Fetcher) *DgtleHandler {
	return &DgtleHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *DgtleHandler) GetPath() string {
	return "/dgtle"
}

// Handle 处理请求
func (h *DgtleHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchDgtle(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"dgtle_hot",
		"数字尾巴",
		"热门文章",
		"数字尾巴热门文章列表",
		"https://www.dgtle.com",
		nil,
		data,
		!noCache,
	))
}

// fetchDgtle 从数字尾巴 API 获取数据
func (h *DgtleHandler) fetchDgtle(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://opser.api.dgtle.com/v2/news/index"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求数字尾巴 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp DgtleAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析数字尾巴响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Items), nil
}

// transformData 将数字尾巴原始数据转换为统一格式
func (h *DgtleHandler) transformData(items []DgtleItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 标题优先使用 title,否则使用 content
		title := item.Title
		if title == "" {
			title = item.Content
		}

		// 时间戳转换
		timestamp := strconv.FormatInt(item.CreatedAt, 10)

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     title,
			Desc:      item.Content,
			Cover:     item.Cover,
			Author:    item.From,
			Hot:       item.Membernum,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.dgtle.com/news-%d-%d.html", item.ID, item.Type),
			MobileURL: fmt.Sprintf("https://m.dgtle.com/news-details/%d", item.ID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是数字尾巴 API 的响应结构体定义

// DgtleAPIResponse 数字尾巴 API 响应
type DgtleAPIResponse struct {
	Items []DgtleItem `json:"items"`
}

// DgtleItem 单个文章项
type DgtleItem struct {
	ID        int64  `json:"id"`         // 文章 ID
	Title     string `json:"title"`      // 标题
	Content   string `json:"content"`    // 内容
	Cover     string `json:"cover"`      // 封面图
	From      string `json:"from"`       // 来源/作者
	Membernum int64  `json:"membernum"`  // 热度(成员数)
	CreatedAt int64  `json:"created_at"` // 创建时间(时间戳)
	Type      int    `json:"type"`       // 类型
}
