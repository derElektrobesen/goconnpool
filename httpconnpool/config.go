package httpconnpool

import (
	"time"

	"github.com/derElektrobesen/goconnpool"
)

const (
	// Forever describes a value used to disable any timeout in this package.
	Forever = 365 * 24 * time.Hour
)

const (
	// DefaultRequestTimeout declares default value for GetConnectionTimeout config variable
	DefaultRequestTimeout = 5 * time.Second

	// DefaultGetConnectionTimeout declares default value for GetConnectionTimeout config variable
	DefaultGetConnectionTimeout = 10 * time.Second
)

// Config describes a configuration of the http connections pool
type Config struct {
	// PoolConfig describes a configuration for protocol-independent connections pool
	PoolConfig *goconnpool.Config

	// GetConnectionTimeout describes a timeout to get a connection from the connections pool.
	// You could set Forever value for this variable if you want wait for connection forever.
	GetConnectionTimeout time.Duration

	// RequestTimeout describes a timeout to read/write request into connection.
	// You could set Forever value for this variable to disable this timeout.
	RequestTimeout time.Duration
}

// NewConfig creates new configuration for the http connections pool from flags parser passed.
// See FlagsParsed description for more info: https://godoc.org/github.com/derElektrobesen/goconnpool#FlagsParser
func NewConfig(p goconnpool.FlagsParser) *Config {
	c := &Config{
		PoolConfig: goconnpool.NewConfig(p),
	}

	p.DurationVar(&c.RequestTimeout, "request_timeout",
		DefaultRequestTimeout, "HTTP request timeout")
	p.DurationVar(&c.GetConnectionTimeout, "get_connection_from_pool_timeout",
		DefaultGetConnectionTimeout, "Timeout to get connection from pool")

	return c
}

// WithDefaults returns a copy of the base configuration object with filled with defaults not set config variables.
func (c Config) WithDefaults() Config {
	if c.PoolConfig == nil {
		c.PoolConfig = &goconnpool.Config{}
	}

	baseCfg := c.PoolConfig.WithDefaults()
	c.PoolConfig = &baseCfg

	if c.RequestTimeout == 0 {
		c.RequestTimeout = DefaultRequestTimeout
	}

	if c.GetConnectionTimeout == 0 {
		c.GetConnectionTimeout = DefaultGetConnectionTimeout
	}

	return c
}
