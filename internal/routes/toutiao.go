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

// ToutiaoHandler 今日头条处理器
type ToutiaoHandler struct {
	fetcher *service.Fetcher
}

// NewToutiaoHandler 创建今日头条处理器
func NewToutiaoHandler(fetcher *service.Fetcher) *ToutiaoHandler {
	return &ToutiaoHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *ToutiaoHandler) GetPath() string {
	return "/toutiao"
}

// Handle 处理请求
func (h *ToutiaoHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchToutiaoHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"toutiao",                  // name: 平台调用名称
		"今日头条",                     // title: 平台显示名称
		"热榜",                       // type: 榜单类型
		"发现今日头条热门资讯",               // description: 平台描述
		"https://www.toutiao.com/", // link: 官方链接
		nil,                        // params: 无参数映射
		data,                       // data: 热榜数据
		!noCache,                   // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchToutiaoHot 从今日头条 API 获取数据
func (h *ToutiaoHandler) fetchToutiaoHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://www.toutiao.com/hot-event/hot-board/?origin=toutiao_pc"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求今日头条 API 失败: %w", err)
	}

	var apiResp ToutiaoAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析今日头条响应失败: %w", err)
	}

	return h.transformData(apiResp.Data), nil
}

// transformData 转换数据格式
func (h *ToutiaoHandler) transformData(items []ToutiaoItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 从 ClusterIdStr 提取时间戳(前10位是秒级时间戳)
		timestamp := item.ClusterIdStr
		if len(timestamp) > 10 {
			timestamp = timestamp[:10]
		}

		// 处理HotValue字段(可能是int64或string)
		var hotValue int64
		switch v := item.HotValue.(type) {
		case float64:
			hotValue = int64(v)
		case int64:
			hotValue = v
		case string:
			// 字符串转int64
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				hotValue = parsed
			}
		default:
			hotValue = 0
		}

		hotData := models.HotData{
			ID:        item.ClusterIdStr,
			Title:     item.Title,
			Cover:     item.Image.URL,
			Hot:       hotValue,
			URL:       fmt.Sprintf("https://www.toutiao.com/trending/%s/", item.ClusterIdStr),
			MobileURL: fmt.Sprintf("https://api.toutiaoapi.com/feoffline/amos_land/new/html/main/index.html?topic_id=%s", item.ClusterIdStr),
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result
}

// ToutiaoAPIResponse 今日头条 API 响应
type ToutiaoAPIResponse struct {
	Data []ToutiaoItem `json:"data"`
}

// ToutiaoItem 热点项
type ToutiaoItem struct {
	ClusterIdStr string       `json:"ClusterIdStr"`
	Title        string       `json:"Title"`
	Image        ToutiaoImage `json:"Image"`
	HotValue     interface{}  `json:"HotValue"` // 可能是int64或string
}

// ToutiaoImage 图片信息
type ToutiaoImage struct {
	URL string `json:"url"`
}
