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

// YystvHandler 游研社处理器
type YystvHandler struct {
	fetcher *service.Fetcher
}

// NewYystvHandler 创建游研社处理器
func NewYystvHandler(fetcher *service.Fetcher) *YystvHandler {
	return &YystvHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *YystvHandler) GetPath() string {
	return "/yystv"
}

// Handle 处理请求
func (h *YystvHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchYystv(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		"yystv_docs",
		"游研社",
		"全部文章",
		"游研社全部文章",
		"https://www.yystv.cn",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchYystv 从游研社 API 获取数据
func (h *YystvHandler) fetchYystv(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.yystv.cn/home/get_home_docs_by_page"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求游研社 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp YystvAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析游研社响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data), nil
}

// transformData 将游研社原始数据转换为统一格式
func (h *YystvHandler) transformData(items []YystvItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理ID字段(可能是int64或string)
		var itemID int64
		var itemIDStr string
		switch v := item.ID.(type) {
		case float64:
			itemID = int64(v)
			itemIDStr = strconv.FormatInt(int64(v), 10)
		case int64:
			itemID = v
			itemIDStr = strconv.FormatInt(v, 10)
		case string:
			itemIDStr = v
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				itemID = parsed
			}
		default:
			itemID = 0
			itemIDStr = "0"
		}

		// 处理CreateTime字段(可能是int64或string)
		var timestamp string
		switch v := item.CreateTime.(type) {
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
			ID:        itemIDStr,
			Title:     item.Title,
			Cover:     item.Cover,
			Author:    item.Author,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.yystv.cn/p/%d", itemID),
			MobileURL: fmt.Sprintf("https://www.yystv.cn/p/%d", itemID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是游研社 API 的响应结构体定义

// YystvAPIResponse 游研社 API 响应
type YystvAPIResponse struct {
	Data []YystvItem `json:"data"`
}

// YystvItem 单个文章项
type YystvItem struct {
	ID         interface{} `json:"id"`         // 文章 ID (可能是int64或string)
	Title      string      `json:"title"`      // 标题
	Cover      string      `json:"cover"`      // 封面图
	Author     string      `json:"author"`     // 作者
	CreateTime interface{} `json:"createtime"` // 创建时间 (可能是int64或string)
}
