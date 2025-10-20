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

// Kr36Handler 36氪处理器
type Kr36Handler struct {
	fetcher *service.Fetcher
}

// NewKr36Handler 创建36氪处理器
func NewKr36Handler(fetcher *service.Fetcher) *Kr36Handler {
	return &Kr36Handler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *Kr36Handler) GetPath() string {
	return "/36kr"
}

// Handle 处理请求
func (h *Kr36Handler) Handle(c *fiber.Ctx) error {
	rankType := c.Query("type", "hot")
	noCache := c.Query("cache") == "false"
	typeMap := map[string]string{
		"hot":     "人气榜",
		"video":   "视频榜",
		"comment": "热议榜",
		"collect": "收藏榜",
	}
	typeName := typeMap[rankType]
	if typeName == "" {
		typeName = "人气榜"
	}
	data, err := h.fetchKr36Hot(c.Context(), rankType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}
	resp := models.SuccessResponse(
		"36kr", "36氪", typeName, "发现36氪热门资讯",
		"https://36kr.com/", map[string]interface{}{"type": typeMap},
		data, !noCache,
	)
	return c.JSON(resp)
}

// fetchKr36Hot 从36氪 API 获取数据
func (h *Kr36Handler) fetchKr36Hot(ctx context.Context, rankType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://gateway.36kr.com/api/mis/nav/home/nav/rank/%s", rankType)

	// 构造 POST 请求体
	requestBody := map[string]interface{}{
		"partner_id": "wap",
		"param": map[string]interface{}{
			"siteId":     1,
			"platformId": 2,
		},
		"timestamp": time.Now().UnixMilli(),
	}

	bodyBytes, _ := json.Marshal(requestBody)

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Post(apiURL, bodyBytes, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	})
	if err != nil {
		return nil, fmt.Errorf("请求36氪 API 失败: %w", err)
	}

	var apiResp Kr36APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析36氪响应失败: %w", err)
	}

	// 根据类型获取对应的列表
	var list []Kr36Item
	switch rankType {
	case "hot":
		list = apiResp.Data.HotRankList
	case "video":
		list = apiResp.Data.VideoList
	case "comment":
		list = apiResp.Data.RemarkList
	case "collect":
		list = apiResp.Data.CollectList
	default:
		list = apiResp.Data.HotRankList
	}

	return h.transformData(list), nil
}

// transformData 转换数据格式
func (h *Kr36Handler) transformData(items []Kr36Item) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		material := item.TemplateMaterial

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ItemID, 10),
			Title:     material.WidgetTitle,
			Cover:     material.WidgetImage,
			Author:    material.AuthorName,
			Hot:       material.StatCollect,
			URL:       fmt.Sprintf("https://www.36kr.com/p/%d", item.ItemID),
			MobileURL: fmt.Sprintf("https://m.36kr.com/p/%d", item.ItemID),
			Timestamp: item.PublishTime * 1000, // 时间戳转换为毫秒级
		}

		result = append(result, hotData)
	}

	return result
}

// Kr36APIResponse 36氪 API 响应
type Kr36APIResponse struct {
	Data Kr36Data `json:"data"`
}

// Kr36Data 数据部分
type Kr36Data struct {
	HotRankList []Kr36Item `json:"hotRankList"`
	VideoList   []Kr36Item `json:"videoList"`
	RemarkList  []Kr36Item `json:"remarkList"`
	CollectList []Kr36Item `json:"collectList"`
}

// Kr36Item 文章项
type Kr36Item struct {
	ItemID           int64                `json:"itemId"`
	TemplateMaterial Kr36TemplateMaterial `json:"templateMaterial"`
	PublishTime      int64                `json:"publishTime"`
}

// Kr36TemplateMaterial 模板素材
type Kr36TemplateMaterial struct {
	WidgetTitle string `json:"widgetTitle"`
	WidgetImage string `json:"widgetImage"`
	AuthorName  string `json:"authorName"`
	StatCollect int64  `json:"statCollect"`
}
