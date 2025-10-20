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

// ThePaperHandler 澎湃新闻处理器
type ThePaperHandler struct {
	fetcher *service.Fetcher
}

// NewThePaperHandler 创建澎湃新闻处理器
func NewThePaperHandler(fetcher *service.Fetcher) *ThePaperHandler {
	return &ThePaperHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *ThePaperHandler) GetPath() string {
	return "/thepaper"
}

// Handle 处理请求
func (h *ThePaperHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchThePaper(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"thepaper",                 // name: 平台调用名称
		"澎湃新闻",                     // title: 平台显示名称
		"热榜",                       // type: 榜单类型
		"发现澎湃新闻热门资讯",               // description: 平台描述
		"https://www.thepaper.cn/", // link: 官方链接
		nil,                        // params: 无参数映射
		data,                       // data: 热榜数据
		!noCache,                   // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchThePaper 从澎湃新闻 API 获取数据
func (h *ThePaperHandler) fetchThePaper(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://cache.thepaper.cn/contentapi/wwwIndex/rightSidebar"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求澎湃新闻 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp ThePaperAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析澎湃新闻响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.HotNews), nil
}

// transformData 将澎湃新闻原始数据转换为统一格式
func (h *ThePaperHandler) transformData(items []ThePaperItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理PraiseTimes字段(可能是int64或string)
		var praiseTimes int64
		switch v := item.PraiseTimes.(type) {
		case float64:
			praiseTimes = int64(v)
		case int64:
			praiseTimes = v
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				praiseTimes = parsed
			}
		default:
			praiseTimes = 0
		}

		// 转换时间戳(毫秒 -> 秒)
		timestamp := strconv.FormatInt(item.PubTimeLong/1000, 10)

		hotData := models.HotData{
			ID:        item.ContID,
			Title:     item.Name,
			Cover:     item.Pic,
			Hot:       praiseTimes,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.thepaper.cn/newsDetail_forward_%s", item.ContID),
			MobileURL: fmt.Sprintf("https://m.thepaper.cn/newsDetail_forward_%s", item.ContID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是澎湃新闻 API 的响应结构体定义

// ThePaperAPIResponse 澎湃新闻 API 响应
type ThePaperAPIResponse struct {
	Data ThePaperData `json:"data"`
}

// ThePaperData 数据部分
type ThePaperData struct {
	HotNews []ThePaperItem `json:"hotNews"`
}

// ThePaperItem 单个新闻项
type ThePaperItem struct {
	ContID      string      `json:"contId"`      // 内容 ID
	Name        string      `json:"name"`        // 标题
	Pic         string      `json:"pic"`         // 封面图
	PraiseTimes interface{} `json:"praiseTimes"` // 点赞数 (可能是int64或string)
	PubTimeLong int64       `json:"pubTimeLong"` // 发布时间(毫秒)
}
