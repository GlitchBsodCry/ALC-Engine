package middleware

import (
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"mygo_bangforai/api/errors"
)

type RateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex			// 读写锁
	r   rate.Limit 				// 每秒填充的令牌数
	b   int 					// 令牌桶的容量
}

func NewIPRateLimiter(r rate.Limit, b int) *RateLimiter {// 创建一个新的IP速率限制器
    return &RateLimiter{
        ips: make(map[string]*rate.Limiter),
        mu:  &sync.RWMutex{},
        r:   r,
        b:   b,
    }
}

func (i *RateLimiter) AddIP(ip string) *rate.Limiter {// 添加一个IP到速率限制器
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)
    i.ips[ip] = limiter
    return limiter
}

func (i *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if exists {
		return limiter
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists = i.ips[ip]
	if exists {
		return limiter
	}

	limiter = rate.NewLimiter(i.r, i.b)
	i.ips[ip] = limiter
	return limiter
}


func RateLimitMiddleware(r rate.Limit, b int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(r,b)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		lim := limiter.GetLimiter(ip)
		if !lim.Allow() {
			errors.Error(c, errors.TooManyRequests, "IP rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}