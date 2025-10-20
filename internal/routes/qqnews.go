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

// QQNewsHandler 腾讯新闻处理器
type QQNewsHandler struct {
	fetcher *service.Fetcher
}

// NewQQNewsHandler 创建腾讯新闻处理器
func NewQQNewsHandler(fetcher *service.Fetcher) *QQNewsHandler {
	return &QQNewsHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *QQNewsHandler) GetPath() string {
	return "/qq-news"
}

// Handle 处理请求
func (h *QQNewsHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchQQNews(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"qq-news",              // name: 平台调用名称
		"腾讯新闻",                 // title: 平台显示名称
		"热点榜",                  // type: 榜单类型
		"发现腾讯新闻热门资讯",           // description: 平台描述
		"https://news.qq.com/", // link: 官方链接
		nil,                    // params: 无参数映射
		data,                   // data: 热榜数据
		!noCache,               // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchQQNews 从腾讯新闻 API 获取数据
func (h *QQNewsHandler) fetchQQNews(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://r.inews.qq.com/gw/event/hot_ranking_list?page_size=50"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求腾讯新闻 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp QQNewsAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析腾讯新闻响应失败: %w", err)
	}

	// 检查数据
	if len(apiResp.IDList) == 0 || len(apiResp.IDList[0].NewsList) == 0 {
		return nil, fmt.Errorf("腾讯新闻数据为空")
	}

	// 跳过第一个(通常是广告或置顶)
	newsList := apiResp.IDList[0].NewsList
	if len(newsList) > 1 {
		newsList = newsList[1:]
	}

	// 转换为统一格式
	return h.transformData(newsList), nil
}

// transformData 将腾讯新闻原始数据转换为统一格式
func (h *QQNewsHandler) transformData(items []QQNewsItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 转换时间戳(秒)
		timestamp := strconv.FormatInt(item.Timestamp, 10)

		hotData := models.HotData{
			ID:        item.ID,
			Title:     item.Title,
			Desc:      item.Abstract,
			Cover:     item.MiniProShareImage,
			Author:    item.Source,
			Hot:       item.HotEvent.HotScore,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://new.qq.com/rain/a/%s", item.ID),
			MobileURL: fmt.Sprintf("https://view.inews.qq.com/k/%s", item.ID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是腾讯新闻 API 的响应结构体定义

// QQNewsAPIResponse 腾讯新闻 API 响应
type QQNewsAPIResponse struct {
	IDList []QQNewsIDList `json:"idlist"`
}

// QQNewsIDList ID 列表
type QQNewsIDList struct {
	NewsList []QQNewsItem `json:"newslist"`
}

// QQNewsItem 单个新闻项
type QQNewsItem struct {
	ID                string         `json:"id"`                // 新闻 ID
	Title             string         `json:"title"`             // 标题
	Abstract          string         `json:"abstract"`          // 摘要
	MiniProShareImage string         `json:"miniProShareImage"` // 封面图
	Source            string         `json:"source"`            // 来源
	Timestamp         int64          `json:"timestamp"`         // 时间戳(秒)
	HotEvent          QQNewsHotEvent `json:"hotEvent"`          // 热度信息
}

// QQNewsHotEvent 热度事件
type QQNewsHotEvent struct {
	HotScore int64 `json:"hotScore"` // 热度分数
}
