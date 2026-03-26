package breaker

import (
	"time"

	"github.com/chenzanhong/zlog"
	"github.com/sony/gobreaker"
)

// CircuitBreakerManager 熔断器管理器
type CircuitBreakerManager struct {
	smtpBreaker *gobreaker.CircuitBreaker
}

// NewCircuitBreakerManager 创建熔断器管理器
func NewCircuitBreakerManager() *CircuitBreakerManager {
	// 配置 SMTP 服务的熔断器
	smtpSettings := gobreaker.Settings{
		Name:        "smtp-service",
		MaxRequests: 3,                // 半开状态下允许的最大请求数
		Interval:    30 * time.Second, // 统计窗口时间
		Timeout:     60 * time.Second, // 熔断后多久尝试半开状态
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// 失败率超过 30% 时触发熔断
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// 状态变化时的回调，可以添加日志
			zlog.Infof("Circuit breaker %s changed from %v to %v", name, from, to)
		},
	}

	return &CircuitBreakerManager{
		smtpBreaker: gobreaker.NewCircuitBreaker(smtpSettings),
	}
}

// ExecuteWithBreaker 使用熔断器执行函数
func (c *CircuitBreakerManager) ExecuteWithBreaker(breaker *gobreaker.CircuitBreaker, fn func() (interface{}, error)) (interface{}, error) {
	return breaker.Execute(func() (interface{}, error) {
		return fn()
	})
}

// GetSMTPBreaker 获取 SMTP 服务的熔断器
func (c *CircuitBreakerManager) GetSMTPBreaker() *gobreaker.CircuitBreaker {
	return c.smtpBreaker
}
