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

// GenshinHandler 原神处理器
type GenshinHandler struct {
	fetcher *service.Fetcher
}

// NewGenshinHandler 创建原神处理器
func NewGenshinHandler(fetcher *service.Fetcher) *GenshinHandler {
	return &GenshinHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *GenshinHandler) GetPath() string {
	return "/genshin"
}

// Handle 处理请求
func (h *GenshinHandler) Handle(c *fiber.Ctx) error {
	newsType := c.Query("type", "1") // 默认公告
	noCache := c.Query("cache") == "false"

	data, err := h.fetchGenshin(c.Context(), newsType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		fmt.Sprintf("genshin_%s", newsType),
		"原神",
		"最新动态",
		"原神最新动态列表",
		"https://www.miyoushe.com/ys",
		nil,
		data,
		!noCache,
	))
}

// fetchGenshin 从米游社 API 获取原神数据
func (h *GenshinHandler) fetchGenshin(ctx context.Context, newsType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?client_type=4&gids=2&last_id=&page_size=20&type=%s", newsType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求米游社 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp MiyousheAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析米游社响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.List, "ys"), nil
}

// transformData 将米游社原始数据转换为统一格式
func (h *GenshinHandler) transformData(items []MiyousheListItem, game string) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		post := item.Post

		// 获取封面图
		cover := post.Cover
		if cover == "" && len(post.Images) > 0 {
			cover = post.Images[0]
		}

		// 作者昵称
		author := ""
		if item.User != nil {
			author = item.User.Nickname
		}

		// 时间戳转换
		timestamp := strconv.FormatInt(post.CreatedAt, 10)

		hotData := models.HotData{
			ID:        post.PostID,
			Title:     post.Subject,
			Desc:      post.Content,
			Cover:     cover,
			Author:    author,
			Hot:       post.ViewStatus,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.miyoushe.com/%s/article/%s", game, post.PostID),
			MobileURL: fmt.Sprintf("https://m.miyoushe.com/%s/#/article/%s", game, post.PostID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是米游社 API 的响应结构体定义(共用)

// MiyousheAPIResponse 米游社 API 响应
type MiyousheAPIResponse struct {
	Data MiyousheData `json:"data"`
}

// MiyousheData 数据部分
type MiyousheData struct {
	List []MiyousheListItem `json:"list"`
}

// MiyousheListItem 列表项
type MiyousheListItem struct {
	Post MiyoushePost  `json:"post"`
	User *MiyousheUser `json:"user"`
}

// MiyoushePost 帖子信息
type MiyoushePost struct {
	PostID     string   `json:"post_id"`     // 帖子 ID
	Subject    string   `json:"subject"`     // 标题
	Content    string   `json:"content"`     // 内容
	Cover      string   `json:"cover"`       // 封面图
	Images     []string `json:"images"`      // 图片列表
	ViewStatus int64    `json:"view_status"` // 浏览量
	CreatedAt  int64    `json:"created_at"`  // 创建时间
}

// MiyousheUser 用户信息
type MiyousheUser struct {
	Nickname string `json:"nickname"` // 昵称
}
