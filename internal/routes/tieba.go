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

// TiebaHandler 百度贴吧处理器
type TiebaHandler struct {
	fetcher *service.Fetcher
}

// NewTiebaHandler 创建百度贴吧处理器
func NewTiebaHandler(fetcher *service.Fetcher) *TiebaHandler {
	return &TiebaHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *TiebaHandler) GetPath() string {
	return "/tieba"
}

// Handle 处理请求
func (h *TiebaHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchTieba(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		"tieba_hot",
		"百度贴吧",
		"热议榜",
		"百度贴吧热议榜",
		"https://tieba.baidu.com/hottopic/browse/topicList",
		nil,
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// fetchTieba 从百度贴吧 API 获取数据
func (h *TiebaHandler) fetchTieba(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://tieba.baidu.com/hottopic/browse/topicList"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求百度贴吧 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp TiebaAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析百度贴吧响应失败: %w", err)
	}

	// 检查数据结构
	if apiResp.Data.BangTopic.TopicList == nil {
		return nil, fmt.Errorf("百度贴吧数据为空")
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.BangTopic.TopicList), nil
}

// transformData 将百度贴吧原始数据转换为统一格式
func (h *TiebaHandler) transformData(items []TiebaItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 转换时间戳
		timestamp := strconv.FormatInt(item.CreateTime, 10)

		// 处理TopicID字段（可能是int64或string）
		topicID := ""
		switch v := item.TopicID.(type) {
		case float64:
			topicID = strconv.FormatInt(int64(v), 10)
		case int64:
			topicID = strconv.FormatInt(v, 10)
		case string:
			topicID = v
		default:
			topicID = ""
		}

		hotData := models.HotData{
			ID:        topicID,
			Title:     item.TopicName,
			Desc:      item.TopicDesc,
			Cover:     item.TopicPic,
			Hot:       item.DiscussNum,
			Timestamp: timestamp,
			URL:       item.TopicURL,
			MobileURL: item.TopicURL,
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是百度贴吧 API 的响应结构体定义

// TiebaAPIResponse 百度贴吧 API 响应
type TiebaAPIResponse struct {
	Data TiebaData `json:"data"`
}

// TiebaData 数据部分
type TiebaData struct {
	BangTopic TiebaBangTopic `json:"bang_topic"`
}

// TiebaBangTopic 榜单话题
type TiebaBangTopic struct {
	TopicList []TiebaItem `json:"topic_list"`
}

// TiebaItem 单个话题项
type TiebaItem struct {
	TopicID    interface{} `json:"topic_id"`    // 话题 ID（可能是int64或string）
	TopicName  string      `json:"topic_name"`  // 话题名称
	TopicDesc  string      `json:"topic_desc"`  // 话题描述
	TopicPic   string      `json:"topic_pic"`   // 话题图片
	DiscussNum int64       `json:"discuss_num"` // 讨论数
	CreateTime int64       `json:"create_time"` // 创建时间
	TopicURL   string      `json:"topic_url"`   // 话题链接
}
