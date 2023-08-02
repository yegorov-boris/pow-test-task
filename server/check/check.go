package check

import (
	"sync"
	"time"
)

type Checker struct {
	cache sync.Map
	ttl   time.Duration
}

func NewChecker(ttl time.Duration) *Checker {
	c := &Checker{
		ttl: ttl,
	}
	s := 1 * time.Second
	if c.ttl < s {
		c.ttl = s
	}

	return c
}

func (c *Checker) Run(done <-chan struct{}) {
	ticker := time.NewTicker(c.ttl)
	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case now := <-ticker.C:
				c.cache.Range(func(key, value any) bool {
					if now.Sub(value.(time.Time)) > c.ttl {
						c.cache.Delete(key)
					}

					return true
				})
			}
		}
	}()
}

func (c *Checker) RecentlyUsed(id []byte) bool {
	_, found := c.cache.LoadOrStore(string(id), time.Now())

	return found
}

func (c *Checker) CheckPoW(zeros uint8, hashSum []byte) bool {
	fullyZeroBytes := int(zeros / 8)
	if len(hashSum) < fullyZeroBytes+1 {
		return false
	}

	for i := 0; i < fullyZeroBytes; i++ {
		if hashSum[i] != 0 {
			return false
		}
	}

	lastZeros := zeros % 8

	return lastZeros == 0 || (hashSum[fullyZeroBytes]>>(8-lastZeros)) == 0
}
