package routes

import (
	"context"
	"fmt"
	"time"

	"github.com/dailyhot/api/internal/http"
)

// RetryConfig 重试配置
// 注意：HTTP 客户端已经内置了 Resty 的自动重试机制
// 本文件中的重试函数主要用于需要特殊重试策略的场景
type RetryConfig struct {
	MaxRetries   int           // 最大重试次数
	InitialDelay time.Duration // 初始延迟时间
	MaxDelay     time.Duration // 最大延迟时间
}

// DefaultRetryConfig 默认重试配置
// 3 次重试，初始延迟 1 秒，最大延迟 5 秒
var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     5 * time.Second,
}

// FetchWithRetry 带重试机制的 HTTP 请求函数
// 实现了指数退避策略，适用于需要更强控制的场景
//
// 参数:
//   - ctx: 上下文，用于超时控制和取消
//   - client: HTTP 客户端
//   - url: 请求 URL
//   - headers: 请求头
//   - config: 重试配置
//
// 返回:
//   - 响应体字节数组
//   - 错误信息
//
// 说明:
// 指数退避策略：
//   - 第 1 次失败后等 1 秒
//   - 第 2 次失败后等 2 秒
//   - 依此类推，不超过 MaxDelay
//
// 与 Resty 内置重试的差异：
// - Resty 使用线性增长 (1s + 1s + 1s)
// - 本函数使用指数增长 (1s + 2s + 3s)，更适合长时间的网络不稳定
func FetchWithRetry(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
	config RetryConfig,
) ([]byte, error) {
	var lastErr error

	// 重试逻辑
	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		// 尝试获取数据
		body, err := client.Get(url, headers)
		if err == nil && len(body) > 0 {
			// 成功获取，直接返回
			return body, nil
		}

		// 保存最后的错误信息
		lastErr = err

		// 如果不是最后一次重试，进行延迟
		if attempt < config.MaxRetries-1 {
			// 计算延迟时间：指数退避
			// 第 1 次失败后等 1 秒
			// 第 2 次失败后等 2 秒
			// 依此类推，不超过 MaxDelay
			delayDuration := time.Duration(attempt+1) * config.InitialDelay
			if delayDuration > config.MaxDelay {
				delayDuration = config.MaxDelay
			}

			// 等待或被取消
			select {
			case <-time.After(delayDuration):
				// 继续重试
			case <-ctx.Done():
				// 上下文已取消，停止重试
				return nil, fmt.Errorf("请求被取消: %w", ctx.Err())
			}
		}
	}

	// 所有重试都失败，返回最后的错误
	return nil, fmt.Errorf("请求失败(已重试 %d 次): %w", config.MaxRetries, lastErr)
}

// FetchWithDefaultRetry 使用默认重试配置的请求函数
// 这是 FetchWithRetry 的便捷包装，用于需要更强控制的平台
// 大多数平台可以直接使用 httpClient.Get()，因为 Resty 已内置重试
func FetchWithDefaultRetry(
	ctx context.Context,
	client *http.Client,
	url string,
	headers map[string]string,
) ([]byte, error) {
	return FetchWithRetry(ctx, client, url, headers, DefaultRetryConfig)
}
