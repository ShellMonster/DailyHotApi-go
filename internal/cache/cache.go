package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/dailyhot/api/internal/config"
	"github.com/dailyhot/api/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Cache 缓存管理器接口
// 定义了缓存的基本操作方法
type Cache interface {
	// Get 获取缓存数据
	Get(ctx context.Context, key string) ([]byte, error)

	// Set 设置缓存数据
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Close 关闭缓存连接
	Close() error
}

// Manager 双层缓存管理器
// 就像一个智能仓库管理系统:
// - L1(BigCache): 超快的本地货架,但容量有限
// - L2(Redis): 稍慢的共享仓库,容量大且可多机共享
type Manager struct {
	l1Cache   *bigcache.BigCache // 第一层:内存缓存(BigCache)
	l2Cache   *redis.Client      // 第二层:Redis 缓存
	cfg       *config.Config     // 配置信息
	l1Enabled bool               // L1 是否启用
	l2Enabled bool               // L2 是否启用
}

// NewManager 创建缓存管理器
func NewManager(cfg *config.Config) (*Manager, error) {
	m := &Manager{
		cfg:       cfg,
		l1Enabled: cfg.Cache.Enabled,
		l2Enabled: cfg.Redis.Enabled,
	}

	// 初始化 L1 缓存 (BigCache)
	if m.l1Enabled {
		if err := m.initL1Cache(); err != nil {
			return nil, fmt.Errorf("初始化 L1 缓存失败: %w", err)
		}
		logger.Info("L1 缓存(BigCache)初始化成功")
	}

	// 初始化 L2 缓存 (Redis)
	if m.l2Enabled {
		if err := m.initL2Cache(); err != nil {
			// Redis 失败不影响整体运行,只记录警告
			logger.Warn("L2 缓存(Redis)初始化失败", zap.Error(err))
			m.l2Enabled = false
		} else {
			logger.Info("L2 缓存(Redis)初始化成功")
		}
	}

	return m, nil
}

// initL1Cache 初始化 BigCache
func (m *Manager) initL1Cache() error {
	config := bigcache.Config{
		// Shards: 分片数量,增加并发性能
		// 设置为 1024,可以减少锁竞争
		Shards: 1024,

		// LifeWindow: 数据存活时间
		LifeWindow: m.cfg.Cache.DefaultExpire,

		// CleanWindow: 清理过期数据的间隔
		CleanWindow: m.cfg.Cache.CleanupInterval,

		// MaxEntriesInWindow: 时间窗口内的最大条目数
		MaxEntriesInWindow: m.cfg.Cache.MaxEntries * 10,

		// MaxEntrySize: 单个条目最大大小(字节)
		MaxEntrySize: m.cfg.Cache.MaxEntrySize,

		// HardMaxCacheSize: 缓存总大小上限(MB)
		HardMaxCacheSize: m.cfg.Cache.HardMaxCacheSize,

		// Verbose: 是否输出详细日志
		Verbose: false,
	}

	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		return err
	}

	m.l1Cache = cache
	return nil
}

// initL2Cache 初始化 Redis
func (m *Manager) initL2Cache() error {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", m.cfg.Redis.Host, m.cfg.Redis.Port),
		Password:     m.cfg.Redis.Password,
		DB:           m.cfg.Redis.DB,
		PoolSize:     m.cfg.Redis.PoolSize,
		DialTimeout:  m.cfg.Redis.Timeout,
		ReadTimeout:  m.cfg.Redis.Timeout,
		WriteTimeout: m.cfg.Redis.Timeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis 连接失败: %w", err)
	}

	m.l2Cache = client
	return nil
}

// Get 获取缓存数据
// 智能查找流程:
// 1. 先查 L1(内存),超快
// 2. L1 没有,查 L2(Redis)
// 3. L2 有数据,回填到 L1,下次更快
func (m *Manager) Get(ctx context.Context, key string) ([]byte, error) {
	// 1. 尝试从 L1 获取
	if m.l1Enabled {
		data, err := m.l1Cache.Get(key)
		if err == nil {
			// L1 命中,直接返回
			logger.Debug("L1 缓存命中", zap.String("key", key))
			return data, nil
		}
	}

	// 2. L1 未命中,尝试从 L2 获取
	if m.l2Enabled {
		data, err := m.l2Cache.Get(ctx, key).Bytes()
		if err == nil {
			// L2 命中,回填到 L1
			logger.Debug("L2 缓存命中", zap.String("key", key))
			if m.l1Enabled {
				_ = m.l1Cache.Set(key, data)
			}
			return data, nil
		}
		if err != redis.Nil {
			// Redis 错误(非 key 不存在)
			logger.Warn("L2 缓存读取失败", zap.String("key", key), zap.Error(err))
		}
	}

	// 两层缓存都未命中
	return nil, fmt.Errorf("缓存未命中: %s", key)
}

// Set 设置缓存数据
// 同时写入两层缓存,确保数据一致性
func (m *Manager) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	// 如果没指定过期时间,使用默认值
	if expiration == 0 {
		expiration = m.cfg.Cache.DefaultExpire
	}

	// 写入 L1 缓存
	if m.l1Enabled {
		if err := m.l1Cache.Set(key, value); err != nil {
			logger.Warn("L1 缓存写入失败", zap.String("key", key), zap.Error(err))
		} else {
			logger.Debug("L1 缓存写入成功", zap.String("key", key))
		}
	}

	// 写入 L2 缓存
	if m.l2Enabled {
		if err := m.l2Cache.Set(ctx, key, value, expiration).Err(); err != nil {
			logger.Warn("L2 缓存写入失败", zap.String("key", key), zap.Error(err))
		} else {
			logger.Debug("L2 缓存写入成功", zap.String("key", key))
		}
	}

	return nil
}

// Delete 删除缓存
// 同时删除两层缓存
func (m *Manager) Delete(ctx context.Context, key string) error {
	// 删除 L1 缓存
	if m.l1Enabled {
		if err := m.l1Cache.Delete(key); err != nil {
			logger.Warn("L1 缓存删除失败", zap.String("key", key), zap.Error(err))
		}
	}

	// 删除 L2 缓存
	if m.l2Enabled {
		if err := m.l2Cache.Del(ctx, key).Err(); err != nil {
			logger.Warn("L2 缓存删除失败", zap.String("key", key), zap.Error(err))
		}
	}

	return nil
}

// Close 关闭缓存连接
// 程序退出时调用,释放资源
func (m *Manager) Close() error {
	// 关闭 L1
	if m.l1Enabled && m.l1Cache != nil {
		if err := m.l1Cache.Close(); err != nil {
			logger.Error("关闭 L1 缓存失败", zap.Error(err))
		}
	}

	// 关闭 L2
	if m.l2Enabled && m.l2Cache != nil {
		if err := m.l2Cache.Close(); err != nil {
			logger.Error("关闭 L2 缓存失败", zap.Error(err))
		}
	}

	logger.Info("缓存系统已关闭")
	return nil
}

// GetStats 获取缓存统计信息
// 用于监控缓存性能
func (m *Manager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if m.l1Enabled && m.l1Cache != nil {
		l1Stats := m.l1Cache.Stats()
		stats["l1"] = map[string]interface{}{
			"hits":       l1Stats.Hits,
			"misses":     l1Stats.Misses,
			"del_hits":   l1Stats.DelHits,
			"del_misses": l1Stats.DelMisses,
			"collisions": l1Stats.Collisions,
		}
	}

	if m.l2Enabled && m.l2Cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		info, err := m.l2Cache.Info(ctx, "stats").Result()
		if err == nil {
			stats["l2"] = map[string]interface{}{
				"info": info,
			}
		}
	}

	return stats
}
