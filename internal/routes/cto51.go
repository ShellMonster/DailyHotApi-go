package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

// CTO51Handler 51CTO处理器
type CTO51Handler struct {
	fetcher     *service.Fetcher
	token       string    // 缓存token
	tokenExpire time.Time // token过期时间
}

// NewCTO51Handler 创建51CTO处理器
func NewCTO51Handler(fetcher *service.Fetcher) *CTO51Handler {
	return &CTO51Handler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *CTO51Handler) GetPath() string {
	return "/51cto"
}

// Handle 处理请求
func (h *CTO51Handler) Handle(c *fiber.Ctx) error {
	noCache := c.Query("cache") == "false"
	data, err := h.fetch51CTOHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}
	resp := models.SuccessResponse(
		"51cto", "51CTO", "推荐榜", "发现51CTO热门资讯",
		"https://www.51cto.com/", nil, data, !noCache,
	)
	return c.JSON(resp)
}

// fetch51CTOHot 从51CTO API 获取数据
func (h *CTO51Handler) fetch51CTOHot(ctx context.Context) ([]models.HotData, error) {
	// 获取或刷新token
	if h.token == "" || time.Now().After(h.tokenExpire) {
		token, err := utils.Get51CTOToken()
		if err != nil {
			return nil, fmt.Errorf("获取51CTO Token失败: %w", err)
		}
		h.token = token
		h.tokenExpire = time.Now().Add(24 * time.Hour) // token有效期24小时
	}

	// 构建请求参数
	requestPath := "index/index/recommend"
	params := map[string]interface{}{
		"page":       1,
		"page_size":  50,
		"limit_time": 0,
		"name_en":    "",
	}

	timestamp := time.Now().UnixMilli()
	sign := utils.Sign51CTO(requestPath, params, timestamp, h.token)

	// 构建完整URL
	apiURL := fmt.Sprintf(
		"https://api-media.51cto.com/index/index/recommend?page=%d&page_size=%d&limit_time=%d&name_en=%s&timestamp=%d&token=%s&sign=%s",
		params["page"], params["page_size"], params["limit_time"], params["name_en"],
		timestamp, h.token, sign,
	)

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求51CTO API 失败: %w", err)
	}

	var apiResp CTO51APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析51CTO响应失败: %w", err)
	}

	return h.transformData(apiResp.Data.Data.List), nil
}

// transformData 转换数据格式
func (h *CTO51Handler) transformData(items []CTO51Item) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理Pubdate字段（可能是int64或string）
		timestamp := ""
		switch v := item.Pubdate.(type) {
		case float64:
			timestamp = strconv.FormatInt(int64(v), 10)
		case int64:
			timestamp = strconv.FormatInt(v, 10)
		case string:
			timestamp = v
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.SourceID, 10),
			Title:     item.Title,
			Cover:     item.Cover,
			Desc:      item.Abstract,
			URL:       item.URL,
			MobileURL: item.URL,
			Timestamp: timestamp,
		}

		result = append(result, hotData)
	}

	return result
}

// CTO51APIResponse 51CTO API 响应
type CTO51APIResponse struct {
	Data CTO51Data `json:"data"`
}

// CTO51Data 数据部分
type CTO51Data struct {
	Data CTO51DataInner `json:"data"`
}

// CTO51DataInner 内层数据
type CTO51DataInner struct {
	List []CTO51Item `json:"list"`
}

// CTO51Item 文章项
type CTO51Item struct {
	SourceID int64       `json:"source_id"`
	Title    string      `json:"title"`
	Cover    string      `json:"cover"`
	Abstract string      `json:"abstract"`
	Pubdate  interface{} `json:"pubdate"` // 可能是int64或string
	URL      string      `json:"url"`
}
