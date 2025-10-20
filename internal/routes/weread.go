package routes

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WereadHandler 微信读书处理器
type WereadHandler struct {
	fetcher *service.Fetcher
}

// NewWereadHandler 创建微信读书处理器
func NewWereadHandler(fetcher *service.Fetcher) *WereadHandler {
	return &WereadHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *WereadHandler) GetPath() string {
	return "/weread"
}

// Handle 处理请求
func (h *WereadHandler) Handle(c *fiber.Ctx) error {
	rankType := c.Query("type", "rising") // 默认飙升榜
	noCache := c.Query("cache") == "false"

	// 直接调用fetch函数获取数据
	data, err := h.fetchWeread(c.Context(), rankType)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建响应
	resp := models.SuccessResponse(
		fmt.Sprintf("weread_%s", rankType),
		"微信读书",
		h.getTypeName(rankType),
		"微信读书热门榜单",
		"https://weread.qq.com",
		map[string]interface{}{"type": rankType},
		data,
		!noCache,
	)

	return c.JSON(resp)
}

// getTypeName 获取榜单类型名称
func (h *WereadHandler) getTypeName(typeID string) string {
	typeMap := map[string]string{
		"rising":               "飙升榜",
		"hot_search":           "热搜榜",
		"newbook":              "新书榜",
		"general_novel_rising": "小说榜",
		"all":                  "总榜",
	}
	if name, ok := typeMap[typeID]; ok {
		return name
	}
	return "飙升榜"
}

// fetchWeread 从微信读书 API 获取数据
func (h *WereadHandler) fetchWeread(ctx context.Context, rankType string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://weread.qq.com/web/bookListInCategory/%s?rank=1", rankType)

	// 发起 HTTP 请求
	httpClient := h.fetcher.GetHTTPClient()
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.67",
	}

	body, err := httpClient.Get(apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("请求微信读书 API 失败: %w", err)
	}

	// 解析 JSON 响应
	var apiResp WereadAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析微信读书响应失败: %w", err)
	}

	// 转换为统一格式
	return h.transformData(apiResp.Books), nil
}

// transformData 将微信读书原始数据转换为统一格式
func (h *WereadHandler) transformData(items []WereadBook) []models.HotData {
	result := make([]models.HotData, 0, len(items))

	for _, item := range items {
		book := item.BookInfo

		// 封面图处理(将 s_ 替换为 t9_)
		cover := strings.Replace(book.Cover, "s_", "t9_", 1)

		// 处理PublishTime字段(可能是int64或string)
		var timestamp string
		switch v := book.PublishTime.(type) {
		case float64:
			timestamp = strconv.FormatInt(int64(v), 10)
		case int64:
			timestamp = strconv.FormatInt(v, 10)
		case string:
			timestamp = v
		default:
			timestamp = ""
		}

		// 获取书籍 ID 的 Base64 编码形式
		bookID := h.getWereadID(book.BookID)

		hotData := models.HotData{
			ID:        book.BookID,
			Title:     book.Title,
			Desc:      book.Intro,
			Cover:     cover,
			Author:    book.Author,
			Hot:       item.ReadingCount,
			Timestamp: timestamp,
			URL:       fmt.Sprintf("https://weread.qq.com/web/bookDetail/%s", bookID),
			MobileURL: fmt.Sprintf("https://weread.qq.com/web/bookDetail/%s", bookID),
		}

		result = append(result, hotData)
	}

	return result
}

// getWereadID 将 bookId 转换为编码格式
// 参考原TypeScript实现: src/utils/getToken/weread.ts
func (h *WereadHandler) getWereadID(bookID string) string {
	// 使用 MD5 哈希算法
	hash := md5.Sum([]byte(bookID))
	str := fmt.Sprintf("%x", hash)

	// 取哈希结果的前三个字符作为初始值
	strSub := str[:3]

	// 判断书籍 ID 的类型并进行转换
	var fa []interface{}
	isDigit := true
	for _, ch := range bookID {
		if ch < '0' || ch > '9' {
			isDigit = false
			break
		}
	}

	if isDigit {
		// 如果书籍 ID 只包含数字，则将其拆分成长度为 9 的子字符串，并转换为十六进制表示
		chunks := []string{}
		for i := 0; i < len(bookID); i += 9 {
			end := i + 9
			if end > len(bookID) {
				end = len(bookID)
			}
			chunk := bookID[i:end]
			num, _ := strconv.ParseInt(chunk, 10, 64)
			chunks = append(chunks, fmt.Sprintf("%x", num))
		}
		fa = []interface{}{"3", chunks}
	} else {
		// 如果书籍 ID 包含其他字符，则将每个字符的 Unicode 编码转换为十六进制表示
		hexStr := ""
		for _, ch := range bookID {
			hexStr += fmt.Sprintf("%x", ch)
		}
		fa = []interface{}{"4", []string{hexStr}}
	}

	// 将类型添加到初始值中
	strSub += fa[0].(string)
	// 将数字 2 和哈希结果的后两个字符添加到初始值中
	strSub += "2" + str[len(str)-2:]

	// 处理转换后的子字符串数组
	chunks := fa[1].([]string)
	for i, sub := range chunks {
		subLength := fmt.Sprintf("%x", len(sub))
		// 如果长度只有一位数，则在前面添加 0
		if len(subLength) == 1 {
			subLength = "0" + subLength
		}
		// 将长度和子字符串添加到初始值中
		strSub += subLength + sub
		// 如果不是最后一个子字符串，则添加分隔符 'g'
		if i < len(chunks)-1 {
			strSub += "g"
		}
	}

	// 如果初始值长度不足 20，从哈希结果中取足够的字符补齐
	if len(strSub) < 20 {
		strSub += str[:20-len(strSub)]
	}

	// 使用 MD5 哈希算法创建新的哈希对象
	finalHash := md5.Sum([]byte(strSub))
	finalStr := fmt.Sprintf("%x", finalHash)
	// 取最终哈希结果的前三个字符并添加到初始值的末尾
	strSub += finalStr[:3]

	return strSub
}

// 以下是微信读书 API 的响应结构体定义

// WereadAPIResponse 微信读书 API 响应
type WereadAPIResponse struct {
	Books []WereadBook `json:"books"`
}

// WereadBook 书籍信息
type WereadBook struct {
	BookInfo     WereadBookInfo `json:"bookInfo"`     // 书籍详情
	ReadingCount int64          `json:"readingCount"` // 阅读人数
}

// WereadBookInfo 书籍详情
type WereadBookInfo struct {
	BookID      string      `json:"bookId"`      // 书籍 ID
	Title       string      `json:"title"`       // 标题
	Author      string      `json:"author"`      // 作者
	Intro       string      `json:"intro"`       // 简介
	Cover       string      `json:"cover"`       // 封面图
	PublishTime interface{} `json:"publishTime"` // 发布时间 (可能是int64或string)
}
