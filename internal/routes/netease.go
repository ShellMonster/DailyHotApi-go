package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// NeteaseHandler 网易新闻处理器
type NeteaseHandler struct {
	fetcher *service.Fetcher
}

// NewNeteaseHandler 创建网易新闻处理器
func NewNeteaseHandler(fetcher *service.Fetcher) *NeteaseHandler {
	return &NeteaseHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *NeteaseHandler) GetPath() string {
	return "/netease-news"
}

// Handle 处理请求
func (h *NeteaseHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchNeteaseHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"netease-news",          // name: 平台调用名称
		"网易新闻",                  // title: 平台显示名称
		"热点榜",                   // type: 榜单类型
		"发现网易新闻热门资讯",            // description: 平台描述
		"https://news.163.com/", // link: 官方链接
		nil,                     // params: 无参数映射
		data,                    // data: 热榜数据
		!noCache,                // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchNeteaseHot 从网易新闻 API 获取数据
func (h *NeteaseHandler) fetchNeteaseHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://m.163.com/fe/api/hot/news/flow"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求网易新闻 API 失败: %w", err)
	}

	var apiResp NeteaseAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析网易新闻响应失败: %w", err)
	}

	return h.transformData(apiResp.Data.List), nil
}

// transformData 转换数据格式
func (h *NeteaseHandler) transformData(items []NeteaseItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 解析时间戳(格式: "2024-01-01 12:00:00")
		timestamp := ""
		if item.PTime != "" {
			t, err := time.Parse("2006-01-02 15:04:05", item.PTime)
			if err == nil {
				timestamp = strconv.FormatInt(t.Unix(), 10)
			}
		}

		hotData := models.HotData{
			ID:        item.DocID,
			Title:     item.Title,
			Cover:     item.ImgSrc,
			Author:    item.Source,
			URL:       fmt.Sprintf("https://www.163.com/dy/article/%s.html", item.DocID),
			MobileURL: fmt.Sprintf("https://m.163.com/dy/article/%s.html", item.DocID),
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result
}

// NeteaseAPIResponse 网易新闻 API 响应
type NeteaseAPIResponse struct {
	Data NeteaseData `json:"data"`
}

// NeteaseData 数据部分
type NeteaseData struct {
	List []NeteaseItem `json:"list"`
}

// NeteaseItem 新闻项
type NeteaseItem struct {
	DocID  string `json:"docid"`
	Title  string `json:"title"`
	ImgSrc string `json:"imgsrc"`
	Source string `json:"source"`
	PTime  string `json:"ptime"`
}
