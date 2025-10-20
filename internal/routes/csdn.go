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

// CSDNHandler CSDN处理器
type CSDNHandler struct {
	fetcher *service.Fetcher
}

// NewCSDNHandler 创建CSDN处理器
func NewCSDNHandler(fetcher *service.Fetcher) *CSDNHandler {
	return &CSDNHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *CSDNHandler) GetPath() string {
	return "/csdn"
}

// Handle 处理请求
func (h *CSDNHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchCSDNHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"csdn",                   // name: 平台调用名称
		"CSDN",                   // title: 平台显示名称
		"排行榜",                    // type: 榜单类型
		"发现CSDN热门博文",             // description: 平台描述
		"https://blog.csdn.net/", // link: 官方链接
		nil,                      // params: 无参数映射
		data,                     // data: 热榜数据
		!noCache,                 // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchCSDNHot 从CSDN API 获取数据
func (h *CSDNHandler) fetchCSDNHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://blog.csdn.net/phoenix/web/blog/hot-rank?page=0&pageSize=30"

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求CSDN API 失败: %w", err)
	}

	var apiResp CSDNAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析CSDN响应失败: %w", err)
	}

	return h.transformData(apiResp.Data), nil
}

// transformData 转换数据格式
func (h *CSDNHandler) transformData(items []CSDNItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 封面图片取第一张
		cover := ""
		if len(item.PicList) > 0 {
			cover = item.PicList[0]
		}

		// 处理HotRankScore字段(可能是int64或string)
		var hotRankScore int64
		switch v := item.HotRankScore.(type) {
		case float64:
			hotRankScore = int64(v)
		case int64:
			hotRankScore = v
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				hotRankScore = parsed
			}
		default:
			hotRankScore = 0
		}

		// 处理Period字段(可能是int64或string)
		var timestamp string
		switch v := item.Period.(type) {
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
			ID:        item.ProductID,
			Title:     item.ArticleTitle,
			Cover:     cover,
			Author:    item.NickName,
			Hot:       hotRankScore,
			URL:       item.ArticleDetailURL,
			MobileURL: item.ArticleDetailURL,
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result
}

// CSDNAPIResponse CSDN API 响应
type CSDNAPIResponse struct {
	Data []CSDNItem `json:"data"`
}

// CSDNItem 文章项
type CSDNItem struct {
	ProductID        string      `json:"productId"`
	ArticleTitle     string      `json:"articleTitle"`
	ArticleDetailURL string      `json:"articleDetailUrl"`
	NickName         string      `json:"nickName"`
	HotRankScore     interface{} `json:"hotRankScore"` // 可能是int64或string
	PicList          []string    `json:"picList"`
	Period           interface{} `json:"period"` // 可能是int64或string
}
