package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dailyhot/api/internal/cache"
	"github.com/dailyhot/api/internal/http"
	"github.com/dailyhot/api/internal/logger"
	"github.com/dailyhot/api/internal/models"
	"github.com/dailyhot/api/internal/pool"
	"go.uber.org/zap"
)

// Fetcher 数据获取服务
// 负责协调缓存和 HTTP 请求,提供统一的数据获取接口
type Fetcher struct {
	cache      *cache.Manager   // 缓存管理器
	httpClient *http.Client     // HTTP 客户端
	objectPool *pool.ObjectPool // 对象池管理器(用于内存优化)
}

// NewFetcher 创建数据获取服务
func NewFetcher(cacheManager *cache.Manager) *Fetcher {
	return &Fetcher{
		cache:      cacheManager,
		httpClient: http.GetDefaultClient(),
		objectPool: pool.NewObjectPool(), // 初始化对象池
	}
}

// FetchFunc 数据获取函数类型
// 定义了如何从原始 API 获取并解析数据
// 返回: 热榜数据列表和错误
type FetchFunc func(ctx context.Context) ([]models.HotData, error)

// GetData 获取热榜数据(带缓存)
// 这是核心方法,实现了完整的缓存逻辑
//
// 工作流程:
// 1. 先查缓存,有就直接返回
// 2. 缓存没有,调用 fetchFunc 获取原始数据
// 3. 将数据写入缓存
// 4. 返回数据
//
// 参数:
//   - ctx: 上下文
//   - cacheKey: 缓存键,如 "bilibili_hot"
//   - platformName: 平台名称,如 "哔哩哔哩"
//   - subtitle: 副标题,如 "热门榜"
//   - cacheDuration: 缓存时长,如 5*time.Minute
//   - fetchFunc: 数据获取函数
func (f *Fetcher) GetData(
	ctx context.Context,
	cacheKey string,
	platformName string,
	subtitle string,
	cacheDuration time.Duration,
	fetchFunc FetchFunc,
) (*models.Response, error) {
	// 1. 尝试从缓存获取
	cachedData, err := f.cache.Get(ctx, cacheKey)
	if err == nil {
		// 缓存命中,反序列化数据
		var hotDataList []models.HotData
		if err := json.Unmarshal(cachedData, &hotDataList); err == nil {
			logger.Info("缓存命中",
				zap.String("platform", platformName),
				zap.String("cache_key", cacheKey),
				zap.Int("count", len(hotDataList)),
			)
			// 使用 SimpleSuccessResponse 保持向后兼容
			return models.SimpleSuccessResponse(platformName, subtitle, hotDataList, true), nil
		}
		logger.Warn("缓存数据反序列化失败", zap.Error(err))
	}

	// 2. 缓存未命中,调用 fetchFunc 获取原始数据
	logger.Info("缓存未命中,从源获取数据",
		zap.String("platform", platformName),
		zap.String("cache_key", cacheKey),
	)

	hotDataList, err := fetchFunc(ctx)
	if err != nil {
		logger.Error("获取数据失败",
			zap.String("platform", platformName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("获取 %s 数据失败: %w", platformName, err)
	}

	// 3. 将数据写入缓存
	if len(hotDataList) > 0 {
		dataBytes, err := json.Marshal(hotDataList)
		if err == nil {
			_ = f.cache.Set(ctx, cacheKey, dataBytes, cacheDuration)
			logger.Info("数据已缓存",
				zap.String("platform", platformName),
				zap.String("cache_key", cacheKey),
				zap.Int("count", len(hotDataList)),
			)
		}
	}

	// 4. 返回数据
	// 使用 SimpleSuccessResponse 保持向后兼容
	return models.SimpleSuccessResponse(platformName, subtitle, hotDataList, false), nil
}

// GetHTTPClient 获取 HTTP 客户端
// 供路由处理器使用
func (f *Fetcher) GetHTTPClient() *http.Client {
	return f.httpClient
}

// InvalidateCache 使缓存失效
// 用于手动清除某个平台的缓存
func (f *Fetcher) InvalidateCache(ctx context.Context, cacheKey string) error {
	return f.cache.Delete(ctx, cacheKey)
}

// GetCacheStats 获取缓存统计信息
func (f *Fetcher) GetCacheStats() map[string]interface{} {
	return f.cache.GetStats()
}

// GetObjectPool 获取对象池管理器
// 用于 HTTP 客户端和其他组件使用
func (f *Fetcher) GetObjectPool() *pool.ObjectPool {
	return f.objectPool
}
