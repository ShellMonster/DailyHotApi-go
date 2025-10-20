package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WeiboHandler 微博热搜处理器
type WeiboHandler struct {
	fetcher *service.Fetcher
}

// NewWeiboHandler 创建微博处理器
func NewWeiboHandler(fetcher *service.Fetcher) *WeiboHandler {
	return &WeiboHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *WeiboHandler) GetPath() string {
	return "/weibo"
}

// Handle 处理请求
func (h *WeiboHandler) Handle(c *fiber.Ctx) error {
	// 获取缓存标志
	noCache := c.Query("cache") == "false"

	// 获取热搜数据
	data, err := h.fetchWeiboHot(c.Context())
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"weibo",                           // name: 平台调用名称
		"微博",                              // title: 平台显示名称
		"热搜榜",                             // type: 榜单类型
		"发现微博实时热门话题",                      // description: 平台描述
		"https://s.weibo.com/top/summary", // link: 官方链接
		nil,                               // params: 无参数映射
		data,                              // data: 热榜数据
		!noCache,                          // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchWeiboHot 从微博 API 获取热搜数据
func (h *WeiboHandler) fetchWeiboHot(ctx context.Context) ([]models.HotData, error) {
	// 微博热搜 API
	apiURL := "https://m.weibo.cn/api/container/getIndex?containerid=106003type%3D25%26t%3D3%26disable_hot%3D1%26filter_type%3Drealtimehot&title=%E5%BE%AE%E5%8D%9A%E7%83%AD%E6%90%9C&extparam=filter_type%3Drealtimehot%26mi_cid%3D100103%26pos%3D0_0%26c_type%3D30%26display_time%3D1540538388&luicode=10000011&lfid=231583"

	// 发起 HTTP 请求(需要特定的 User-Agent 模拟移动端)
	// Cookie来源: https://github.com/teg1c/weibo-hot-crawler
	// 感谢 teg1c 提供的微博Cookie解决方案
	httpClient := h.fetcher.GetHTTPClient()
	body, err := httpClient.Get(apiURL, map[string]string{
		"Referer":          "https://s.weibo.com/top/summary?cate=realtimehot",
		"MWeibo-Pwa":       "1",
		"X-Requested-With": "XMLHttpRequest",
		"User-Agent":       "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
		"Cookie":           "SUB=_2AkMWPIzuf8NxqwJRmPgSxGrlZI1xwwvEieKgYH01JRMxHRl-yT8Xqn0HtRB6PbyiAY7W0wZkwFc1nXHJxUddZr9bpaPQ; SUBP=0033WrSXqPxfM72-Ws9jqgMF55529P9D9W5p6Ee5xHahOckML_sA0l2c;",
	})
	if err != nil {
		return nil, fmt.Errorf("请求微博 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp WeiboAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析微博响应失败: %w", err)
	}

	// 检查响应状态
	if apiResp.Ok != 1 {
		return nil, fmt.Errorf("微博 API 返回错误")
	}

	// 检查数据结构
	if len(apiResp.Data.Cards) == 0 || apiResp.Data.Cards[0].CardGroup == nil {
		return nil, fmt.Errorf("微博数据为空")
	}

	// 转换为统一格式
	return h.transformData(apiResp.Data.Cards[0].CardGroup), nil
}

// transformData 将微博原始数据转换为统一格式
func (h *WeiboHandler) transformData(items []WeiboItem) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		// 跳过第一个(通常是置顶广告)
		if item.Desc == "" {
			continue
		}

		// 构造话题关键词
		key := item.WordScheme
		if key == "" {
			key = fmt.Sprintf("#%s", item.Desc)
		}

		hotData := models.HotData{
			ID:    item.ItemID,
			Title: item.Desc,
			Desc:  key,
			// 微博热搜的 URL 是搜索链接
			URL:       fmt.Sprintf("https://s.weibo.com/weibo?q=%s&t=31&band_rank=1&Refer=top", url.QueryEscape(key)),
			Hot:       item.Num, // 热度值
			MobileURL: item.Scheme,

			// 可选字段
			Timestamp: item.OnboardTime * 1000, // 时间戳转换为毫秒级
		}

		result = append(result, hotData)
	}

	return result
}

// 以下是微博 API 的响应结构体定义

// WeiboAPIResponse 微博 API 响应
type WeiboAPIResponse struct {
	Ok   int       `json:"ok"`
	Data WeiboData `json:"data"`
}

// WeiboData 数据部分
type WeiboData struct {
	Cards []WeiboCard `json:"cards"`
}

// WeiboCard 卡片
type WeiboCard struct {
	CardGroup []WeiboItem `json:"card_group"`
}

// WeiboItem 单个热搜项
type WeiboItem struct {
	ItemID      string `json:"itemid"`       // 唯一标识
	Desc        string `json:"desc"`         // 热搜标题
	WordScheme  string `json:"word_scheme"`  // 话题关键词
	OnboardTime int64  `json:"onboard_time"` // 上榜时间
	Num         int64  `json:"num"`          // 热度值
	Scheme      string `json:"scheme"`       // 移动端链接
}
