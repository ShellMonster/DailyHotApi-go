package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/dailyhot/api/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

// BilibiliHandler B站热榜处理器
type BilibiliHandler struct {
	fetcher *service.Fetcher
}

// NewBilibiliHandler 创建 B站处理器
func NewBilibiliHandler(fetcher *service.Fetcher) *BilibiliHandler {
	return &BilibiliHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *BilibiliHandler) GetPath() string {
	return "/bilibili"
}

// Handle 处理请求
func (h *BilibiliHandler) Handle(c *fiber.Ctx) error {
	// 获取 type 参数 (分区ID)
	typeParam := c.Query("type", "0")

	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 构建缓存键
	cacheKey := fmt.Sprintf("bilibili_hot_%s", typeParam)

	// 获取数据
	data, err := h.fetchBilibiliHot(c.Context(), typeParam, noCache)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 获取类型映射
	typeMap := map[string]string{
		"0":   "全站",
		"1":   "动画",
		"3":   "音乐",
		"4":   "游戏",
		"5":   "娱乐",
		"188": "科技",
		"119": "鬼畜",
		"129": "舞蹈",
		"155": "时尚",
		"160": "生活",
		"168": "国创相关",
		"181": "影视",
	}

	// 获取类型显示名称
	typeName := typeMap[typeParam]
	if typeName == "" {
		typeName = "全站"
	}

	// 构建完整的响应 (向后兼容原项目)
	resp := models.SuccessResponse(
		"bilibili",                                    // name: 平台调用名称
		"哔哩哔哩",                                        // title: 平台显示名称
		fmt.Sprintf("热榜 · %s", typeName),              // type: 榜单类型
		"你所热爱的，就是你的生活",                                // description: 平台描述
		"https://www.bilibili.com/v/popular/rank/all", // link: 官方链接
		map[string]interface{}{ // params: 参数说明
			"type": typeMap,
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存 (简化处理)
	)

	_ = cacheKey // 避免未使用警告,实际应该用缓存键

	return c.JSON(resp)
}

// fetchBilibiliHot 从 B站 API 获取热榜数据(双接口策略)
func (h *BilibiliHandler) fetchBilibiliHot(ctx context.Context, typeParam string, noCache bool) ([]models.HotData, error) {
	// 策略1: 尝试主接口(ranking/v2)
	data, err := h.tryMainAPI(ctx, typeParam)
	if err == nil && len(data) > 0 {
		return data, nil
	}

	// 策略2: 主接口失败或无数据,尝试备用接口
	return h.tryBackupAPI(ctx, typeParam)
}

// tryMainAPI 尝试主接口(使用WBI签名的ranking/v2接口)
func (h *BilibiliHandler) tryMainAPI(ctx context.Context, typeParam string) ([]models.HotData, error) {
	// 1. 获取 WBI 签名所需的密钥
	imgKey, subKey, err := utils.GetNavInfo()
	if err != nil {
		return nil, fmt.Errorf("获取 WBI 密钥失败: %w", err)
	}

	// 2. 准备请求参数(使用ranking/v2接口的参数)
	params := map[string]string{
		"rid":  typeParam, // 分区ID(由调用方传入)
		"type": "all",     // 类型
	}

	// 3. 对参数进行 WBI 签名
	signedQuery := utils.EncodeWBI(params, imgKey, subKey)

	// 4. 构造完整 URL
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/ranking/v2?%s", signedQuery)

	// 5. 发起 HTTP 请求(添加完整的浏览器请求头)
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"Referer":            "https://www.bilibili.com/ranking/all",
		"User-Agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
		"Accept":             "application/json, text/plain, */*",
		"Accept-Language":    "zh-CN,zh;q=0.9,en;q=0.8",
		"Sec-Ch-Ua":          `"Google Chrome";v="123", "Not:A-Brand";v="8", "Chromium";v="123"`,
		"Sec-Ch-Ua-Mobile":   "?0",
		"Sec-Ch-Ua-Platform": `"Windows"`,
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-site",
		"Origin":             "https://www.bilibili.com",
	})
	if err != nil {
		return nil, fmt.Errorf("请求 B站主接口失败: %w", err)
	}

	// 6. 解析 JSON 响应
	var apiResp BilibiliRankingResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 B站主接口响应失败: %w", err)
	}

	// 7. 检查响应状态
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("B站主接口返回错误(code=%d): %s", apiResp.Code, apiResp.Message)
	}

	// 8. 检查是否有数据
	if apiResp.Data.List == nil || len(apiResp.Data.List) == 0 {
		return nil, fmt.Errorf("B站主接口无数据")
	}

	// 9. 转换为统一格式
	return h.transformRankingData(apiResp.Data.List), nil
}

// tryBackupAPI 尝试备用接口(不需要WBI签名的ranking接口)
func (h *BilibiliHandler) tryBackupAPI(ctx context.Context, typeParam string) ([]models.HotData, error) {
	// 构造备用接口URL(使用传入的typeParam)
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/ranking?rid=%s&type=all", typeParam)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"Referer":    "https://www.bilibili.com/ranking/all",
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	})
	if err != nil {
		return nil, fmt.Errorf("请求 B站备用接口失败: %w", err)
	}

	// 检查是否是JSONP格式响应
	bodyStr := string(body)
	if len(bodyStr) > 0 && bodyStr[0] != '{' {
		// 去除JSONP包装: __jp0({...})
		re := regexp.MustCompile(`__jp\d+\((.*)\)`)
		matches := re.FindStringSubmatch(bodyStr)
		if len(matches) > 1 {
			body = []byte(matches[1])
		}
	}

	// 解析 JSON 响应
	var apiResp BilibiliRankingResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 B站备用接口响应失败: %w", err)
	}

	// 检查响应状态
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("B站备用接口返回错误(code=%d): %s", apiResp.Code, apiResp.Message)
	}

	// 检查是否有数据
	if apiResp.Data.List == nil || len(apiResp.Data.List) == 0 {
		return nil, fmt.Errorf("B站备用接口也无数据")
	}

	// 转换为统一格式
	return h.transformRankingData(apiResp.Data.List), nil
}

// transformRankingData 将 B站排行榜数据转换为统一格式 (向后兼容)
func (h *BilibiliHandler) transformRankingData(items []BilibiliRankingItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 构造BVID(如果是aid则需要转换)
		bvid := item.BVIDStr
		if bvid == "" {
			bvid = fmt.Sprintf("av%d", item.Aid)
		}

		// 热度值: 使用播放量作为热度 (保持为数字)
		hot := item.Stat.View

		// 时间戳: 转换为毫秒级 (B站 API 返回秒级)
		timestamp := item.Pubdate * 1000

		hotData := models.HotData{
			ID:    bvid, // 使用 BVID 而不是 AID (更符合原项目)
			Title: item.Title,
			Desc:  item.Desc,
			Cover: item.Pic,
			URL:   fmt.Sprintf("https://www.bilibili.com/video/%s", bvid),
			Hot:   hot, // 热度值 (interface{} 类型)

			// 可选字段
			Author:    item.Owner.Name,
			Timestamp: timestamp, // 时间戳转换为毫秒级 (interface{} 类型)
			MobileURL: fmt.Sprintf("https://m.bilibili.com/video/%s", bvid),
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是 B站 API 的响应结构体定义

// BilibiliRankingResponse B站排行榜API响应
type BilibiliRankingResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	TTL     int                 `json:"ttl"`
	Data    BilibiliRankingData `json:"data"`
}

// BilibiliRankingData 排行榜数据部分
type BilibiliRankingData struct {
	List   []BilibiliRankingItem `json:"list"`
	NoMore bool                  `json:"no_more"`
}

// BilibiliRankingItem 排行榜单个视频项
type BilibiliRankingItem struct {
	Aid      int64         `json:"aid"`      // 视频AID
	BVIDStr  string        `json:"bvid"`     // 字符串形式的 BVID
	Title    string        `json:"title"`    // 标题
	Desc     string        `json:"desc"`     // 描述
	Pic      string        `json:"pic"`      // 封面图
	Pubdate  int64         `json:"pubdate"`  // 发布时间戳
	Duration int           `json:"duration"` // 时长(秒)
	Owner    BilibiliOwner `json:"owner"`    // 作者信息
	Stat     BilibiliStat  `json:"stat"`     // 统计信息
	Score    int64         `json:"score"`    // 排行分数
}

// BilibiliOwner 作者信息
type BilibiliOwner struct {
	Mid  int64  `json:"mid"`  // 用户 ID
	Name string `json:"name"` // 用户名
	Face string `json:"face"` // 头像
}

// BilibiliStat 统计信息
type BilibiliStat struct {
	View     int64 `json:"view"`     // 播放量
	Danmaku  int   `json:"danmaku"`  // 弹幕数
	Reply    int   `json:"reply"`    // 评论数
	Favorite int   `json:"favorite"` // 收藏数
	Coin     int   `json:"coin"`     // 投币数
	Share    int   `json:"share"`    // 分享数
	Like     int   `json:"like"`     // 点赞数
}
