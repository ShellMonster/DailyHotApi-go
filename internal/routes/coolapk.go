package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

// CoolapkHandler 酷安处理器
type CoolapkHandler struct {
	fetcher *service.Fetcher
}

// NewCoolapkHandler 创建酷安处理器
func NewCoolapkHandler(fetcher *service.Fetcher) *CoolapkHandler {
	return &CoolapkHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *CoolapkHandler) GetPath() string {
	return "/coolapk"
}

// Handle 处理请求
func (h *CoolapkHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchCoolapkHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"coolapk",                  // name: 平台调用名称
		"酷安",                       // title: 平台显示名称
		"热榜",                       // type: 榜单类型
		"发现酷安平台热门动态",               // description: 平台描述
		"https://www.coolapk.com/", // link: 官方链接
		nil,                        // params: 无参数映射
		data,                       // data: 热榜数据
		!noCache,                   // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchCoolapkHot 从酷安 API 获取数据
func (h *CoolapkHandler) fetchCoolapkHot(ctx context.Context) ([]models.HotData, error) {
	apiURL := "https://api.coolapk.com/v6/page/dataList?url=/feed/statList?cacheExpires=300&statType=day&sortField=detailnum&title=今日热门&title=今日热门&subTitle=&page=1"

	// 生成酷安特殊请求头(包含签名token)
	headers := utils.GenCoolapkHeaders()

	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求酷安 API 失败: %w", err)
	}

	var apiResp CoolapkAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析酷安响应失败: %w", err)
	}

	return h.transformData(apiResp.Data), nil
}

// transformData 转换数据格式
func (h *CoolapkHandler) transformData(items []CoolapkItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		if item.ID == 0 {
			continue
		}

		title := strings.TrimSpace(item.TTitle)
		if title == "" {
			title = strings.TrimSpace(item.Title)
		}
		if title == "" {
			title = strings.TrimSpace(item.Message)
		}
		if title == "" {
			continue
		}

		desc := strings.TrimSpace(item.Message)
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}

		shareURL := strings.TrimSpace(item.ShareURL)
		if shareURL == "" {
			if item.URL != "" {
				if strings.HasPrefix(item.URL, "http") {
					shareURL = item.URL
				} else {
					shareURL = "https://www.coolapk.com" + item.URL
				}
			} else {
				shareURL = fmt.Sprintf("https://www.coolapk.com/feed/%d", item.ID)
			}
		}

		hotData := models.HotData{
			ID:        strconv.FormatInt(item.ID, 10),
			Title:     title,
			Cover:     item.TPic,
			Desc:      desc,
			Author:    item.Username,
			Hot:       int64(item.LikeNum),
			Timestamp: item.Dateline * 1000,
			URL:       shareURL,
			MobileURL: shareURL,
		}

		result = append(result, hotData)
	}

	return result
}

// CoolapkAPIResponse 酷安 API 响应
type CoolapkAPIResponse struct {
	Data []CoolapkItem `json:"data"`
}

// CoolapkItem 动态项
type CoolapkItem struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	TPic     string `json:"tpic"`
	TTitle   string `json:"ttitle"`
	Username string `json:"username"`
	ShareURL string `json:"shareUrl"`
	URL      string `json:"url"`
	Dateline int64  `json:"dateline"`
	LikeNum  int    `json:"likenum"`
}
