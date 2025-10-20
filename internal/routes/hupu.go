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

// HupuHandler 虎扑处理器
type HupuHandler struct {
	fetcher *service.Fetcher
}

// NewHupuHandler 创建虎扑处理器
func NewHupuHandler(fetcher *service.Fetcher) *HupuHandler {
	return &HupuHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *HupuHandler) GetPath() string {
	return "/hupu"
}

// Handle 处理请求
func (h *HupuHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数: 支持不同主题分区 (1-主干道, 6-恋爱区, 11-校园区, 12-历史区, 612-摄影区)
	topicType := c.Query("type", "1")
	noCache := c.Query("cache") == "false"

	// 主题分区映射
	typeMap := map[string]string{
		"1":   "主干道",
		"6":   "恋爱区",
		"11":  "校园区",
		"12":  "历史区",
		"612": "摄影区",
	}

	typeName := typeMap[topicType]
	if typeName == "" {
		typeName = "步行街热帖"
	}

	// 获取数据
	data, err := h.fetchHupuHot(c.Context(), topicType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"hupu",                                  // name: 平台调用名称
		"虎扑",                                    // title: 平台显示名称
		typeName,                                // type: 榜单类型
		"发现虎扑步行街热门帖子",                           // description: 平台描述
		"https://bbs.hupu.com/",                 // link: 官方链接
		map[string]interface{}{"type": typeMap}, // params: 主题分区映射
		data,                                    // data: 热榜数据
		!noCache,                                // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchHupuHot 从虎扑 API 获取数据
func (h *HupuHandler) fetchHupuHot(ctx context.Context, topicType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://m.hupu.com/api/v2/bbs/topicThreads?topicId=%s&page=1", topicType)

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求虎扑 API 失败: %w", err)
	}

	var apiResp HupuAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析虎扑响应失败: %w", err)
	}

	return h.transformData(apiResp.Data.TopicThreads), nil
}

// transformData 转换数据格式
func (h *HupuHandler) transformData(items []HupuItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		hotData := models.HotData{
			ID:        strconv.FormatInt(item.TID, 10),
			Title:     item.Title,
			Author:    item.Username,
			Hot:       int64(item.Replies),
			URL:       fmt.Sprintf("https://bbs.hupu.com/%d.html", item.TID),
			MobileURL: item.URL,
		}

		result = append(result, hotData)
	}

	return result
}

// HupuAPIResponse 虎扑 API 响应
type HupuAPIResponse struct {
	Data HupuData `json:"data"`
}

// HupuData 数据部分
type HupuData struct {
	TopicThreads []HupuItem `json:"topicThreads"`
}

// HupuItem 帖子项
type HupuItem struct {
	TID      int64  `json:"tid"`
	Title    string `json:"title"`
	Username string `json:"username"`
	Replies  int    `json:"replies"`
	URL      string `json:"url"`
}
