package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dailyhot/api/internal/logger"
	"github.com/dailyhot/api/internal/pool"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// Client HTTP 客户端封装
// 基于 Resty 库,提供统一的 HTTP 请求能力
type Client struct {
	client     *resty.Client    // Resty HTTP 客户端
	objectPool *pool.ObjectPool // 对象池管理器(可选,用于性能优化)
}

// NewClient 创建 HTTP 客户端
// 配置了合理的超时、重试等参数
func NewClient() *Client {
	client := resty.New()

	// 基础配置
	client.
		SetTimeout(15 * time.Second).        // 总超时时间 15 秒
		SetRetryCount(3).                    // 失败重试 3 次
		SetRetryWaitTime(1 * time.Second).   // 重试间隔 1 秒
		SetRetryMaxWaitTime(5 * time.Second) // 最大重试等待时间 5 秒

	// 设置请求头
	client.SetHeaders(map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":     "application/json, text/plain, */*",
	})

	// 添加请求拦截器(记录日志)
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		logger.Debug("发起 HTTP 请求",
			zap.String("method", req.Method),
			zap.String("url", req.URL),
		)
		return nil
	})

	// 添加响应拦截器(记录日志和错误)
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		logger.Debug("HTTP 响应",
			zap.String("url", resp.Request.URL),
			zap.Int("status", resp.StatusCode()),
			zap.Duration("time", resp.Time()),
		)
		return nil
	})

	// 添加错误拦截器
	client.OnError(func(req *resty.Request, err error) {
		logger.Warn("HTTP 请求失败",
			zap.String("url", req.URL),
			zap.Error(err),
		)
	})

	return &Client{
		client:     client,
		objectPool: pool.NewObjectPool(), // 初始化对象池以支持缓冲区复用
	}
}

// Get 发起 GET 请求
// url: 请求地址
// headers: 自定义请求头(可选)
// 返回: 响应体字节数组
func (c *Client) Get(url string, headers map[string]string) ([]byte, error) {
	req := c.client.R()

	// 设置自定义请求头
	if headers != nil {
		req.SetHeaders(headers)
	}

	// 发起请求
	resp, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET 请求失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode())
	}

	return resp.Body(), nil
}

// Post 发起 POST 请求
// url: 请求地址
// body: 请求体(JSON 对象或字符串)
// headers: 自定义请求头(可选)
func (c *Client) Post(url string, body interface{}, headers map[string]string) ([]byte, error) {
	req := c.client.R()

	// 设置请求体
	req.SetBody(body)

	// 设置自定义请求头
	if headers != nil {
		req.SetHeaders(headers)
	}

	// 发起请求
	resp, err := req.Post(url)
	if err != nil {
		return nil, fmt.Errorf("POST 请求失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		return nil, fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode())
	}

	return resp.Body(), nil
}

// GetWithCookies 发起带 Cookie 的 GET 请求
// 某些网站需要携带 Cookie 才能访问
func (c *Client) GetWithCookies(url string, cookies map[string]string, headers map[string]string) ([]byte, error) {
	req := c.client.R()

	// 设置 Cookies
	if cookies != nil {
		req.SetCookies(convertCookies(cookies))
	}

	// 设置自定义请求头
	if headers != nil {
		req.SetHeaders(headers)
	}

	// 发起请求
	resp, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET 请求失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode())
	}

	return resp.Body(), nil
}

// GetJSON 发起 GET 请求并解析 JSON
// result: 用于接收解析结果的结构体指针
func (c *Client) GetJSON(url string, result interface{}, headers map[string]string) error {
	req := c.client.R()

	// 设置结果容器
	req.SetResult(result)

	// 设置自定义请求头
	if headers != nil {
		req.SetHeaders(headers)
	}

	// 发起请求
	resp, err := req.Get(url)
	if err != nil {
		return fmt.Errorf("GET 请求失败: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode() != 200 {
		return fmt.Errorf("HTTP 状态码异常: %d", resp.StatusCode())
	}

	return nil
}

// GetWithResponse 发起 GET 请求并返回完整的响应对象（包括响应头）
// 用于需要访问响应头的场景（如获取 Cookie）
func (c *Client) GetWithResponse(url string, headers map[string]string) (*resty.Response, error) {
	req := c.client.R()

	// 设置自定义请求头
	if headers != nil {
		req.SetHeaders(headers)
	}

	// 发起请求
	resp, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET 请求失败: %w", err)
	}

	// 返回完整的响应对象（不检查状态码，由调用方决定）
	return resp, nil
}

// SetTimeout 设置超时时间
// 可以针对特定场景调整超时时间
func (c *Client) SetTimeout(timeout time.Duration) *Client {
	c.client.SetTimeout(timeout)
	return c
}

// SetRetry 设置重试次数
func (c *Client) SetRetry(count int, waitTime time.Duration) *Client {
	c.client.SetRetryCount(count).SetRetryWaitTime(waitTime)
	return c
}

// SetHeader 设置全局请求头
func (c *Client) SetHeader(key, value string) *Client {
	c.client.SetHeader(key, value)
	return c
}

// SetProxy 设置代理
// 某些场景下可能需要通过代理访问
func (c *Client) SetProxy(proxyURL string) *Client {
	c.client.SetProxy(proxyURL)
	return c
}

// GetRawClient 获取原始 Resty 客户端
// 用于一些高级定制场景
func (c *Client) GetRawClient() *resty.Client {
	return c.client
}

// SetObjectPool 设置对象池管理器
// 用于共享对象池以优化内存使用
func (c *Client) SetObjectPool(op *pool.ObjectPool) *Client {
	if op != nil {
		c.objectPool = op
	}
	return c
}

// GetObjectPool 获取对象池管理器
// 用于其他模块共享使用
func (c *Client) GetObjectPool() *pool.ObjectPool {
	return c.objectPool
}

// convertCookies 将 map[string]string 转换为 []*http.Cookie
func convertCookies(cookies map[string]string) []*http.Cookie {
	result := make([]*http.Cookie, 0, len(cookies))
	for name, value := range cookies {
		result = append(result, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	return result
}

// 全局客户端实例
var defaultClient *Client

// GetDefaultClient 获取默认 HTTP 客户端
// 单例模式,全局共享一个客户端实例
func GetDefaultClient() *Client {
	if defaultClient == nil {
		defaultClient = NewClient()
	}
	return defaultClient
}
