package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// LolHandler 英雄联盟处理器
type LolHandler struct {
	fetcher *service.Fetcher
}

// NewLolHandler 创建英雄联盟处理器
func NewLolHandler(fetcher *service.Fetcher) *LolHandler {
	return &LolHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *LolHandler) GetPath() string {
	return "/lol"
}

// Handle 处理请求
func (h *LolHandler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetchLol(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		"lol_news",
		"英雄联盟",
		"更新公告",
		"英雄联盟更新公告列表",
		"https://lol.qq.com",
		nil,
		data,
		!noCache,
	))
}

// fetchLol 从英雄联盟官网 API 获取数据
func (h *LolHandler) fetchLol(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://apps.game.qq.com/cmc/zmMcnTargetContentList?r0=json&page=1&num=30&target=24&source=web_pc"

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求英雄联盟 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp LolAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析英雄联盟响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.Result), nil
}

// transformData 将英雄联盟原始数据转换为统一格式
func (h *LolHandler) transformData(items []LolItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理封面图(添加 https: 前缀)
		cover := item.SIMG
		if cover != "" && cover[0] == '/' {
			cover = "https:" + cover
		}

		// 热度转换
		hot, _ := strconv.ParseInt(item.ITotalPlay, 10, 64)

		// 时间戳
		timestamp := item.SCreated

		// URL 编码 docid
		docID := url.QueryEscape(item.IDocID)

		hotData := models.HotData{
			ID:        item.IDocID,
			Title:     item.STitle,
			Cover:     cover,
			Author:    item.SAuthor,
			Hot:       hot,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://lol.qq.com/news/detail.shtml?docid=%s", docID),
			MobileURL: fmt.Sprintf("https://lol.qq.com/news/detail.shtml?docid=%s", docID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是英雄联盟 API 的响应结构体定义

// LolAPIResponse 英雄联盟 API 响应
type LolAPIResponse struct {
	Data LolData `json:"data"`
}

// LolData 数据部分
type LolData struct {
	Result []LolItem `json:"result"`
}

// LolItem 单个新闻项
type LolItem struct {
	IDocID     string `json:"iDocID"`     // 文档 ID
	STitle     string `json:"sTitle"`     // 标题
	SIMG       string `json:"sIMG"`       // 封面图
	SAuthor    string `json:"sAuthor"`    // 作者
	ITotalPlay string `json:"iTotalPlay"` // 浏览量
	SCreated   string `json:"sCreated"`   // 创建时间
}
