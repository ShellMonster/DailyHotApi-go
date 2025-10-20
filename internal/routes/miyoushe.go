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

// MiyousheHandler 米游社处理器
type MiyousheHandler struct {
	fetcher *service.Fetcher
}

// NewMiyousheHandler 创建米游社处理器
func NewMiyousheHandler(fetcher *service.Fetcher) *MiyousheHandler {
	return &MiyousheHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *MiyousheHandler) GetPath() string {
	return "/miyoushe"
}

// Handle 处理请求
func (h *MiyousheHandler) Handle(c *fiber.Ctx) error {
	game := c.Query("game", "1")     // 默认崩坏3
	newsType := c.Query("type", "1") // 默认公告
	noCache := c.Query("cache") == "false"

	gameName := h.getGameName(game)
	data, err := h.fetchMiyoushe(c.Context(), game, newsType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	return c.JSON(models.SuccessResponse(
		fmt.Sprintf("miyoushe_%s_%s", game, newsType),
		fmt.Sprintf("米游社 · %s", gameName),
		fmt.Sprintf("最新%s", h.getTypeName(newsType)),
		fmt.Sprintf("米游社%s最新%s列表", gameName, h.getTypeName(newsType)),
		"https://www.miyoushe.com",
		nil,
		data,
		!noCache,
	))
}

// getGameName 获取游戏名称
func (h *MiyousheHandler) getGameName(gameID string) string {
	gameMap := map[string]string{
		"1": "崩坏3",
		"2": "原神",
		"3": "崩坏学园2",
		"4": "未定事件簿",
		"5": "大别野",
		"6": "崩坏：星穹铁道",
		"8": "绝区零",
	}
	if name, ok := gameMap[gameID]; ok {
		return name
	}
	return "崩坏3"
}

// getTypeName 获取类型名称
func (h *MiyousheHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"1": "公告",
		"2": "活动",
		"3": "资讯",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "公告"
}

// getGameCode 获取游戏代号(用于 URL)
func (h *MiyousheHandler) getGameCode(gameID string) string {
	gameCodeMap := map[string]string{
		"1": "bh3",
		"2": "ys",
		"3": "bh2",
		"4": "wdy",
		"5": "dby",
		"6": "sr",
		"8": "zzz",
	}
	if code, ok := gameCodeMap[gameID]; ok {
		return code
	}
	return "ys"
}

// fetchMiyoushe 从米游社 API 获取数据
func (h *MiyousheHandler) fetchMiyoushe(ctx context.Context, game, newsType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://bbs-api-static.miyoushe.com/painter/wapi/getNewsList?client_type=4&gids=%s&last_id=&page_size=30&type=%s", game, newsType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("请求米游社 API 失败: %w", err)
	}

	// 解析 JSON 响应(使用 genshin.go 中定义的结构体)
	var apiResp MiyousheAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析米游社响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.List, h.getGameCode(game)), nil
}

// transformData 将米游社原始数据转换为统一格式
func (h *MiyousheHandler) transformData(items []MiyousheListItem, gameCode string) []models.HotData {
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
			URL:       fmt.Sprintf("https://www.miyoushe.com/%s/article/%s", gameCode, post.PostID),
			MobileURL: fmt.Sprintf("https://m.miyoushe.com/%s/#/article/%s", gameCode, post.PostID),
		}

		result = append(result, hotData)
	}

	return result
}
