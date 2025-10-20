package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils/timeutil"
	"github.com/gofiber/fiber/v2"
)

// SmzdmHandler 什么值得买处理器
type SmzdmHandler struct {
	fetcher *service.Fetcher
}

// NewSmzdmHandler 创建什么值得买处理器
func NewSmzdmHandler(fetcher *service.Fetcher) *SmzdmHandler {
	return &SmzdmHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *SmzdmHandler) GetPath() string {
	return "/smzdm"
}

// Handle 处理请求
func (h *SmzdmHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	rankType := c.Query("type", "1") // 默认今日热门
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchSmzdm(c.Context(), rankType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"smzdm",                 // name: 平台调用名称
		"什么值得买",                 // title: 平台显示名称
		h.getTypeName(rankType), // type: 榜单类型
		"发现优质消费资讯与好物推荐",          // description: 平台描述
		"https://www.smzdm.com/", // link: 官方链接
		map[string]interface{}{"type": map[string]string{
			"1": "今日热门", "7": "周热门", "30": "月热门",
		}}, // params: 类型映射
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// getTypeName 获取榜单类型名称
func (h *SmzdmHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"1":  "今日热门",
		"7":  "周热门",
		"30": "月热门",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "今日热门"
}

// fetchSmzdm 从什么值得买 API 获取数据
func (h *SmzdmHandler) fetchSmzdm(ctx context.Context, rankType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://post.smzdm.com/rank/json_more/?unit=%s", rankType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"Accept":           "application/json, text/plain, */*",
		"Referer":          "https://post.smzdm.com/rank/",
		"X-Requested-With": "XMLHttpRequest",
		"Accept-Language":  "zh-CN,zh;q=0.9,en;q=0.8",
		"Sec-Fetch-Mode":   "cors",
		"Sec-Fetch-Site":   "same-origin",
		"Sec-Fetch-Dest":   "empty",
	}
	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求什么值得买 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp SmzdmAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析什么值得买响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data), nil
}

// transformData 将什么值得买原始数据转换为统一格式
func (h *SmzdmHandler) transformData(items []SmzdmItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 热度转换(收藏数)
		hot := parseCollectionCount(item.CollectionCount)

		hotData := models.HotData{
			ID:        item.ArticleID,
			Title:     item.Title,
			Desc:      item.Content,
			Cover:     item.PicURL,
			Author:    item.Nickname,
			Hot:       hot,
			Timestamp: timeutil.ParseTime(item.TimeSort),
			URL:       item.JumpLink,
			MobileURL: item.JumpLink,
		}

		result = append(result, hotData)
	}

	return result
}

func parseCollectionCount(value interface{}) int64 {
	switch v := value.(type) {
	case string:
		if v == "" {
			return 0
		}
		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			return num
		}
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

// 以下是什么值得买 API 的响应结构体定义

// SmzdmAPIResponse 什么值得买 API 响应
type SmzdmAPIResponse struct {
	Data []SmzdmItem `json:"data"`
}

// SmzdmItem 单个文章项
type SmzdmItem struct {
	ArticleID       string      `json:"article_id"`       // 文章 ID
	Title           string      `json:"title"`            // 标题
	Content         string      `json:"content"`          // 内容摘要
	PicURL          string      `json:"pic_url"`          // 封面图
	Nickname        string      `json:"nickname"`         // 作者昵称
	CollectionCount interface{} `json:"collection_count"` // 收藏数(可能为字符串或数字)
	TimeSort        int64       `json:"time_sort"`        // 时间戳
	JumpLink        string      `json:"jump_link"`        // 跳转链接
}
