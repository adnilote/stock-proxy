package counter

import (
	"sync"
	"time"
)

type Counter struct {
	duration  float64 // seconds
	timeFirst time.Time
	count     int64
	mu        *sync.RWMutex
}

func NewCounter(duration float64) *Counter {
	return &Counter{
		duration:  duration,
		timeFirst: time.Now(),
		mu:        &sync.RWMutex{},
	}
}
func (c *Counter) Rate() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.timeFirst).Seconds() > c.duration {
		c.count = 0
	}
	return c.count
}
func (c *Counter) Incr() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.timeFirst).Seconds() > 60 {
		c.timeFirst = time.Now()
		c.count = 0
	}
	c.count++

}
