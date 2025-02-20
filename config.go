package netlistener

import (
	"math"
	"sync"

	"golang.org/x/time/rate"
)

// bandwithConfig is a configuration that holds the global limiters and per connection rate limit values
type bandwithConfig struct {
	// we assume that read and write operations are using separate limiters
	// otherwise we would need to use a single limiter for both
	globalWriteLimiter *rate.Limiter
	perConnWriteLimit  rate.Limit

	globalReadLimiter *rate.Limiter

	// We store perConnReadLimit as a value, because we need to set it for each connection
	// When connection is created, it will read the limit from the parwnt config and set it for its own perConnLimiter
	// In this case we have a single place where perConnLimit is defined
	perConnReadLimit rate.Limit

	// just to be extra safe
	mu sync.RWMutex
}

// Both values are optional, if none of them are set then connection will not be throttled
// We could add additional validation for the negative values, but I am keeping it simple for now
func NewBandwithConfig(globalLimit *int, perConnLimit *int) *bandwithConfig {
	config := &bandwithConfig{}

	config.globalWriteLimiter = rate.NewLimiter(formatRateLimit(globalLimit), formatBurst(globalLimit))
	config.globalReadLimiter = rate.NewLimiter(formatRateLimit(globalLimit), formatBurst(globalLimit))

	config.perConnWriteLimit = formatRateLimit(perConnLimit)
	config.perConnReadLimit = formatRateLimit(perConnLimit)

	return config
}

func (c *bandwithConfig) SetGlobalLimit(globalLimit *int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.globalWriteLimiter == nil {
		c.globalWriteLimiter = rate.NewLimiter(formatRateLimit(globalLimit), formatBurst(globalLimit))
	} else {
		c.globalWriteLimiter.SetLimit(formatRateLimit(globalLimit))
		c.globalWriteLimiter.SetBurst(formatBurst(globalLimit))
	}

	if c.globalReadLimiter == nil {
		c.globalReadLimiter = rate.NewLimiter(formatRateLimit(globalLimit), formatBurst(globalLimit))
	} else {
		c.globalReadLimiter.SetLimit(formatRateLimit(globalLimit))
		c.globalReadLimiter.SetBurst(formatBurst(globalLimit))
	}
}

func (c *bandwithConfig) SetPerConnLimit(perConnLimit *int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.perConnReadLimit = formatRateLimit(perConnLimit)
	c.perConnWriteLimit = formatRateLimit(perConnLimit)
}

func (c *bandwithConfig) PerConnWriteLimit() rate.Limit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.perConnWriteLimit
}

func (c *bandwithConfig) PerConnReadLimit() rate.Limit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.perConnReadLimit
}

func (c *bandwithConfig) GlobalReadLimiter() *rate.Limiter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.globalWriteLimiter
}

func (c *bandwithConfig) GlobalWriteLimiter() *rate.Limiter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.globalWriteLimiter
}

// connectionBandwithConfig is a wrapper around bandwithConfig that allows to set per connection limits, while keeping the global limits.
// Used for connections that are created by the listener
type connectionBandwithConfig struct {
	globalConfig *bandwithConfig

	perConnWriteLimiter *rate.Limiter
	perConnReadLimiter  *rate.Limiter
	mu                  sync.RWMutex
}

func NewConnectionBandwithConfig(bandwithConfig *bandwithConfig) *connectionBandwithConfig {
	config := &connectionBandwithConfig{
		globalConfig: bandwithConfig,
	}

	config.perConnReadLimiter = rate.NewLimiter(bandwithConfig.perConnReadLimit, parseBurstFromRateLimit(bandwithConfig.perConnReadLimit))
	config.perConnWriteLimiter = rate.NewLimiter(bandwithConfig.perConnReadLimit, parseBurstFromRateLimit(bandwithConfig.perConnReadLimit))

	return config
}

func (c *connectionBandwithConfig) SetPerConnWriteLimit(perConnLimit rate.Limit) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.perConnWriteLimiter == nil {
		c.perConnWriteLimiter = rate.NewLimiter(perConnLimit, parseBurstFromRateLimit(perConnLimit))
	} else {
		c.perConnWriteLimiter.SetLimit(perConnLimit)
		c.perConnWriteLimiter.SetBurst(parseBurstFromRateLimit(perConnLimit))
	}
}

func (c *connectionBandwithConfig) SetPerConnReadLimit(perConnLimit rate.Limit) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.perConnReadLimiter == nil {
		c.perConnReadLimiter = rate.NewLimiter(perConnLimit, parseBurstFromRateLimit(perConnLimit))
	} else {
		c.perConnReadLimiter.SetLimit(perConnLimit)
		c.perConnReadLimiter.SetBurst(parseBurstFromRateLimit(perConnLimit))
	}
}

func (c *connectionBandwithConfig) PerConnWriteLimiter() *rate.Limiter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.perConnWriteLimiter
}

func (c *connectionBandwithConfig) PerConnReadLimiter() *rate.Limiter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.perConnReadLimiter
}

func (c *connectionBandwithConfig) PerConnWriteLimit() rate.Limit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.globalConfig.perConnWriteLimit
}

func (c *connectionBandwithConfig) PerConnReadLimit() rate.Limit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.globalConfig.perConnReadLimit
}

func (c *connectionBandwithConfig) GlobalReadLimiter() *rate.Limiter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.globalConfig.globalReadLimiter
}

func (c *connectionBandwithConfig) GlobalWriteLimiter() *rate.Limiter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.globalConfig.globalWriteLimiter
}

func formatRateLimit(limit *int) rate.Limit {
	if limit == nil {
		return rate.Inf
	}

	return rate.Limit(*limit)
}

func formatBurst(limit *int) int {
	if limit == nil {
		return 0
	}

	return *limit
}

func parseBurstFromRateLimit(limit rate.Limit) int {
	if limit == rate.Inf {
		return 0
	}

	// trying to fix integer overflow issue in case if math.MaxInt is passed as a limit
	if limit >= rate.Limit(math.MaxInt) {
		return math.MaxInt
	}

	return int(limit)
}
