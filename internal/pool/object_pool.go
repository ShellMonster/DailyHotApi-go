package pool

import (
	"sync"

	"github.com/dailyhot/api/internal/models"
)

// ObjectPool 对象池管理器
// 用于管理可重用对象，减少内存分配和垃圾回收压力
type ObjectPool struct {
	// HTTP响应缓冲区池
	// 用于存放 HTTP 响应体的字节切片
	responseBufferPool *sync.Pool

	// HotData 切片池
	// 用于存放平台热榜数据列表
	hotDataListPool *sync.Pool
}

// NewObjectPool 创建对象池实例
func NewObjectPool() *ObjectPool {
	return &ObjectPool{
		// 初始化响应缓冲区池
		// 用于存放 HTTP 响应体的字节切片
		// 大多数 API 响应体在 1KB - 100KB 之间
		responseBufferPool: &sync.Pool{
			New: func() interface{} {
				// 预分配 256KB 缓冲区
				// 足够容纳大多数 API 响应
				return make([]byte, 0, 256*1024)
			},
		},

		// 初始化 HotData 列表池
		// 用于存放平台热榜数据列表
		// 大多数平台返回 30-100 条数据
		hotDataListPool: &sync.Pool{
			New: func() interface{} {
				// 预分配容量为 50 的切片
				// 避免频繁扩容
				return make([]models.HotData, 0, 50)
			},
		},
	}
}

// AcquireResponseBuffer 获取响应缓冲区
// 用于接收 HTTP API 响应数据
// 使用完毕后必须调用 ReleaseResponseBuffer 放回
func (p *ObjectPool) AcquireResponseBuffer() []byte {
	// 从池中获取对象，如果池为空则创建新对象
	buf := p.responseBufferPool.Get().([]byte)
	// 重置切片长度为 0，但保留底层容量
	// 这样可以重用底层数组
	return buf[:0]
}

// ReleaseResponseBuffer 释放响应缓冲区
// 将使用完毕的缓冲区放回池中
func (p *ObjectPool) ReleaseResponseBuffer(buf []byte) {
	// 避免放回超大缓冲区（可能是异常情况导致的分配）
	// 只回收容量 <= 512KB 的缓冲区
	if cap(buf) <= 512*1024 {
		p.responseBufferPool.Put(buf)
	}
	// 否则丢弃，由 GC 回收
}

// AcquireHotDataList 获取热数据列表
// 用于收集平台热榜数据
// 使用完毕后必须调用 ReleaseHotDataList 放回
func (p *ObjectPool) AcquireHotDataList() []models.HotData {
	// 从池中获取对象
	list := p.hotDataListPool.Get().([]models.HotData)
	// 重置切片长度为 0，但保留容量
	// 这样可以重用底层数组
	return list[:0]
}

// ReleaseHotDataList 释放热数据列表
// 将使用完毕的列表放回池中
func (p *ObjectPool) ReleaseHotDataList(list []models.HotData) {
	// 避免放回超大列表（可能是异常情况导致的分配）
	// 只回收容量 <= 200 的列表
	if cap(list) <= 200 {
		p.hotDataListPool.Put(list)
	}
	// 否则丢弃，由 GC 回收
}

// Stats 返回对象池统计信息（用于监控）
// 注意：sync.Pool 不提供直接的计数方法
// 这个方法仅用于文档和监控接口预留
func (p *ObjectPool) Stats() map[string]interface{} {
	return map[string]interface{}{
		"status": "active",
		"pools": []string{
			"responseBufferPool",     // HTTP 响应缓冲区池
			"hotDataListPool",        // 热榜数据列表池
		},
	}
}
