package goconnpool

import "time"

const (
	defMaxConnsPerServer   = 1
	defMaxRPS              = 100
	defInitBackoffInterval = 100 * time.Millisecond
	defMaxBackoffInterval  = 30 * time.Second
)

// Config holds some fiels required during new connection pool configuration.
type Config struct {
	// MaxConnsPerServer declares maximum number of opened connections per one registered server.
	// Default value is `defMaxConnsPerServer`.
	MaxConnsPerServer int

	// MaxRPS declares muximum number of requests which could be sent into one server (XXX: not connection)
	// per second.
	// Frankly this value regulates only a number of `OpenConn` calls: goconnpool can't regulate number of real
	// network requests.
	//
	// Default is `defMaxRPS`.
	MaxRPS int

	// ConnectTimeout is the maximum amount of time a dial will wait for
	// a connect to complete. If ConnectDeadline is also set, it may fail
	// earlier.
	//
	// See https://golang.org/pkg/net/#Dialer for more info
	ConnectTimeout time.Duration

	// Backoff configuration is used for exponential backoff mechanism.
	// See https://godoc.org/github.com/cenkalti/backoff#ExponentialBackOff for more info

	InitialBackoffInterval time.Duration
	MaxBackoffInterval     time.Duration

	// Clock could be used to reimplement behaviour of system clock.
	// `SystemClock` by default.
	Clock Clock

	// Logger could be used to view some messages, printed by the library.
	// This messages could contain information about server status, about some errors or something else.
	Logger Logger
}

// FlagsParser is needed to embed goconnpool into production application.
// For example `flag.FlagSet` could be used here (https://golang.org/pkg/flag/#FlagSet) or you could implement
// your own config parser.
type FlagsParser interface {
	IntVar(dst *int, name string, def int, descr string)
	DurationVar(p *time.Duration, name string, value time.Duration, usage string)
}

// NewConfig initializes configuration using `FlagsParser`.
// If you use `flag.FlagSet`-based parsers, config will be filled only after `Parse()` method invoked.
func NewConfig(p FlagsParser) *Config {
	var c Config

	p.IntVar(&c.MaxConnsPerServer, "max_conns_per_server", defMaxConnsPerServer,
		"Maximum number of opened connections per server")
	p.IntVar(&c.MaxRPS, "max_rps", defMaxRPS,
		"Maximim number of requests per one server per second")

	p.DurationVar(&c.ConnectTimeout, "connect_timeout", 0,
		"Maximum amount of time a dial will wait for a connect to complete")

	p.DurationVar(&c.InitialBackoffInterval, "init_backoff_interval", defInitBackoffInterval,
		"Initial backoff interval to retry requests")
	p.DurationVar(&c.MaxBackoffInterval, "max_backoff_interval", defMaxBackoffInterval,
		"Maximum backoff interval to retry requests")

	return &c
}

func (c Config) withDefaults() Config {
	if c.MaxConnsPerServer == 0 {
		c.MaxConnsPerServer = defMaxConnsPerServer
	}

	if c.MaxRPS == 0 {
		c.MaxRPS = defMaxRPS
	}

	if c.InitialBackoffInterval == 0 {
		c.InitialBackoffInterval = defInitBackoffInterval
	}

	if c.MaxBackoffInterval == 0 {
		c.MaxBackoffInterval = defMaxBackoffInterval
	}

	if c.Clock == nil {
		c.Clock = SystemClock{}
	}

	if c.Logger == nil {
		c.Logger = DummyLogger{}
	}

	return c
}
