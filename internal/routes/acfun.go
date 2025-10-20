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

// AcfunHandler AcFun 处理器
type AcfunHandler struct {
	fetcher *service.Fetcher
}

// NewAcfunHandler 创建 AcFun 处理器
func NewAcfunHandler(fetcher *service.Fetcher) *AcfunHandler {
	return &AcfunHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *AcfunHandler) GetPath() string {
	return "/acfun"
}

// Handle 处理请求
func (h *AcfunHandler) Handle(c *fiber.Ctx) error {
	// 获取查询参数
	channelType := c.Query("type", "-1") // 默认综合
	rankRange := c.Query("range", "DAY") // 默认今日
	noCache := c.Query("cache") == "false"

	// 获取数据
	data, err := h.fetchAcfun(c.Context(), channelType, rankRange)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"acfun", // name: 平台调用名称
		"AcFun", // title: 平台显示名称
		fmt.Sprintf("排行榜 · %s", h.getTypeName(channelType)), // type: 榜单类型
		"发现 AcFun 平台热门内容",                                   // description: 平台描述
		"https://www.acfun.cn/",                             // link: 官方链接
		map[string]interface{}{ // params: 参数映射
			"type": map[string]string{
				"-1": "综合", "155": "番剧", "1": "动画",
				"60": "娱乐", "201": "生活", "58": "音乐",
				"123": "舞蹈·偶像", "59": "游戏", "70": "科技",
				"68": "影视", "69": "体育", "125": "鱼塘",
			},
			"range": map[string]string{
				"DAY": "日榜", "WEEK": "周榜", "MONTH": "月榜",
			},
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// getTypeName 获取频道名称
func (h *AcfunHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"-1":  "综合",
		"155": "番剧",
		"1":   "动画",
		"60":  "娱乐",
		"201": "生活",
		"58":  "音乐",
		"123": "舞蹈·偶像",
		"59":  "游戏",
		"70":  "科技",
		"68":  "影视",
		"69":  "体育",
		"125": "鱼塘",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "综合"
}

// fetchAcfun 从 AcFun API 获取数据
func (h *AcfunHandler) fetchAcfun(ctx context.Context, channelType, rankRange string) ([]models.HotData, error) {
	// 处理 channelId(综合频道传空字符串)
	channelID := channelType
	if channelType == "-1" {
		channelID = ""
	}

	apiURL := fmt.Sprintf("https://www.acfun.cn/rest/pc-direct/rank/channel?channelId=%s&rankLimit=30&rankPeriod=%s",
		channelID, rankRange)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"Referer": fmt.Sprintf("https://www.acfun.cn/rank/list/?cid=-1&pcid=%s&range=%s", channelType, rankRange),
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求 AcFun API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp AcfunAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 AcFun 响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.RankList), nil
}

// transformData 将 AcFun 原始数据转换为统一格式
func (h *AcfunHandler) transformData(items []AcfunItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 处理DougaID字段(可能是int64或string)
		var dougaID int64
		var dougaIDStr string
		switch v := item.DougaID.(type) {
		case float64:
			dougaID = int64(v)
			dougaIDStr = strconv.FormatInt(int64(v), 10)
		case int64:
			dougaID = v
			dougaIDStr = strconv.FormatInt(v, 10)
		case string:
			dougaIDStr = v
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				dougaID = parsed
			}
		default:
			dougaID = 0
			dougaIDStr = "0"
		}

		// 时间戳转换
		timestamp := strconv.FormatInt(item.ContributeTime, 10)

		hotData := models.HotData{
			ID:        dougaIDStr,
			Title:     item.ContentTitle,
			Desc:      item.ContentDesc,
			Cover:     item.CoverURL,
			Author:    item.UserName,
			Hot:       item.LikeCount,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://www.acfun.cn/v/ac%d", dougaID),
			MobileURL: fmt.Sprintf("https://m.acfun.cn/v/?ac=%d", dougaID),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是 AcFun API 的响应结构体定义

// AcfunAPIResponse AcFun API 响应
type AcfunAPIResponse struct {
	RankList []AcfunItem `json:"rankList"`
}

// AcfunItem 单个视频项
type AcfunItem struct {
	DougaID        interface{} `json:"dougaId"`        // 视频 ID (可能是int64或string)
	ContentTitle   string      `json:"contentTitle"`   // 标题
	ContentDesc    string      `json:"contentDesc"`    // 描述
	CoverURL       string      `json:"coverUrl"`       // 封面图
	UserName       string      `json:"userName"`       // 作者
	LikeCount      int64       `json:"likeCount"`      // 点赞数
	ContributeTime int64       `json:"contributeTime"` // 投稿时间
}
