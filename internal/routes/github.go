package routes

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/service"
	"github.com/gofiber/fiber/v2"
)

// GitHubHandler GitHub Trending 处理器
type GitHubHandler struct {
	fetcher *service.Fetcher
}

// NewGitHubHandler 创建 GitHub 处理器
func NewGitHubHandler(fetcher *service.Fetcher) *GitHubHandler {
	return &GitHubHandler{
		fetcher: fetcher,
	}
}

// GetPath 获取路由路径
func (h *GitHubHandler) GetPath() string {
	return "/github"
}

// Handle 处理请求
func (h *GitHubHandler) Handle(c *fiber.Ctx) error {
	// 获取类型参数 (daily/weekly/monthly)
	since := c.Query("type", "daily")
	noCache := c.Query("cache") == "false"

	// 类型映射表
	typeMap := map[string]string{
		"daily":   "日榜",
		"weekly":  "周榜",
		"monthly": "月榜",
	}

	// 获取当前时间范围名称
	typeName := typeMap[since]
	if typeName == "" {
		typeName = "日榜"
	}

	// 获取数据
	data, err := h.fetchGitHubTrending(c.Context(), since)
	if err != nil {
		return c.Status(500).JSON(models.ErrorResponseObj(500, err.Error()))
	}

	// 构建完整响应 (向后兼容原项目API格式)
	resp := models.SuccessResponse(
		"github",                               // name: 平台调用名称
		"GitHub",                               // title: 平台显示名称
		fmt.Sprintf("Trending · %s", typeName), // type: 当前时间范围
		"发现GitHub热门开源项目",                       // description: 平台描述
		"https://github.com/trending",          // link: 官方链接
		map[string]interface{}{ // params: 参数说明
			"type": typeMap,
		},
		data,     // data: 热榜数据
		!noCache, // fromCache: 是否来自缓存
	)

	return c.JSON(resp)
}

// fetchGitHubTrending 从 GitHub 获取 Trending 数据(带重试机制)
func (h *GitHubHandler) fetchGitHubTrending(ctx context.Context, since string) ([]models.HotData, error) {
	apiURL := fmt.Sprintf("https://r.jina.ai/https://github.com/trending?since=%s", since)

	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":          "text/plain; charset=utf-8",
		"Accept-Language": "en-US,en;q=0.8",
	}

	// 重试配置
	const maxRetries = 3
	var lastErr error
	httpClient := h.fetcher.GetHTTPClient()

	// 重试逻辑(3 次重试,指数退避)
	for attempt := 0; attempt < maxRetries; attempt++ {
		body, err := httpClient.Get(apiURL, headers)
		if err == nil && len(body) > 0 {
			return h.parseGitHubMarkdown(string(body)), nil
		}

		lastErr = err
		if attempt < maxRetries-1 {
			// 不是最后一次重试,等待后继续
			// 指数退避: 第1次失败等 1 秒,第2次失败等 2 秒
			backoffDuration := time.Duration(attempt+1) * time.Second
			select {
			case <-time.After(backoffDuration):
				// 继续重试
			case <-ctx.Done():
				// 上下文已取消,停止重试
				return nil, fmt.Errorf("GitHub 请求被取消: %w", ctx.Err())
			}
		}
	}

	// 所有重试都失败
	return nil, fmt.Errorf("GitHub 请求失败(重试 %d 次): %w", maxRetries, lastErr)
}

var repoLinePattern = regexp.MustCompile(`^\[(?P<owner>[^/\]]+)\s*/\s*(?P<repo>[^\]]+)\]\((?P<link>https://github\.com/[^\)]+)\)`)
var infoLinePattern = regexp.MustCompile(`^(?P<language>.+?)\[(?P<stars>[\d,]+)\].*?(?P<forks>[\d,]+)\]\([^\)]*\).*?(?P<today>[\d,]+)\s+stars today`)

// parseGitHubMarkdown 解析 Jina 输出的 GitHub Trending Markdown
func (h *GitHubHandler) parseGitHubMarkdown(markdown string) []models.HotData {
	lines := strings.Split(markdown, "\n")
	result := make([]models.HotData, 0, 25)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		matches := repoLinePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		owner := strings.TrimSpace(matches[repoLinePattern.SubexpIndex("owner")])
		repo := strings.TrimSpace(matches[repoLinePattern.SubexpIndex("repo")])
		link := matches[repoLinePattern.SubexpIndex("link")]

		// 向后查找描述与信息行
		desc := ""
		infoLine := ""

		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if next == "" || strings.HasPrefix(next, "[Star]") || strings.HasPrefix(next, "---") {
				continue
			}
			if desc == "" {
				desc = next
				continue
			}
			infoLine = next
			break
		}

		infoMatch := infoLinePattern.FindStringSubmatch(infoLine)
		language := ""
		starsToday := int64(0)
		if infoMatch != nil {
			language = strings.TrimSpace(infoMatch[infoLinePattern.SubexpIndex("language")])
			today := strings.ReplaceAll(infoMatch[infoLinePattern.SubexpIndex("today")], ",", "")
			if v, err := strconv.ParseInt(today, 10, 64); err == nil {
				starsToday = v
			}
		}

		title := fmt.Sprintf("%s/%s", owner, repo)
		if desc == "" {
			desc = title
		}

		hotData := models.HotData{
			ID:        title,
			Title:     title,
			Desc:      desc,
			Author:    owner,
			Cover:     language,
			Hot:       starsToday,
			URL:       link,
			MobileURL: link,
		}

		result = append(result, hotData)

		if len(result) >= 25 {
			break
		}
	}

	return result
}

// stripHTMLTags 移除 HTML 标签
func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}
