package goconnpool

import "time"

const (
	// DefaultMaxConnsPerServer is the default value for MaxConnsPerServer config variable.
	DefaultMaxConnsPerServer = 1

	// DefaultMaxRPS is the default value for MaxRPS config variable.
	DefaultMaxRPS = 100

	// DefaultConnectTimeout is the default value for ConnectTimeout config variable.
	DefaultConnectTimeout = 5 * time.Second

	// DefaultInitBackoffInterval is the default value for InitialBackoffInterval config variable.
	DefaultInitBackoffInterval = 100 * time.Millisecond

	// DefaultMaxBackoffInterval is the default value for MaxBackoffInterval config variable.
	DefaultMaxBackoffInterval = 30 * time.Second
)

// Config holds some fields required during new connection establishing.
type Config struct {
	// MaxConnsPerServer declares maximum number of opened connections per one registered server.
	// Default value is DefaultMaxConnsPerServer.
	//
	// Use math.MaxInt32 to disable this limit.
	MaxConnsPerServer int

	// MaxRPS declares maximum number of requests which could be sent into one server (XXX: not connection)
	// per second.
	// Frankly this value regulates only a number of OpenConn calls: goconnpool can't regulate number of real
	// network requests.
	//
	// Default is DefaultMaxRPS.
	//
	// Use math.MaxInt32 to disable this limit.
	MaxRPS int

	// ConnectTimeout is the maximum amount of time a dial will wait for
	// a connect to complete.
	//
	// Default is DefaultConnectTimeout.
	//
	// Use some large value to disable this timeout.
	ConnectTimeout time.Duration

	// InitialBackoffInterval configures InitialInterval for ExponentialBackOff algorithm.
	// See https://godoc.org/github.com/cenkalti/backoff#ExponentialBackOff for more info.
	//
	// Default is DefaultInitBackoffInterval
	InitialBackoffInterval time.Duration

	// MaxBackoffInterval configures MaxInterval for ExponentialBackOff algorithm.
	// See https://godoc.org/github.com/cenkalti/backoff#ExponentialBackOff for more info.
	//
	// Default is DefaultMaxBackoffInterval
	MaxBackoffInterval time.Duration

	// Clock could be used to reimplement behaviour of system clock.
	// SystemClock by default.
	Clock Clock

	// Logger could be used to view some messages, printed by the library.
	// This messages could contain information about server status, about some errors or something else.
	Logger Logger

	// Dialer is used to dial to the specific server.
	//
	// Required because some protocols (like thrift or websockets) can't be used with raw net.Conn object:
	// they establishes connection in some specific way.
	//
	// TCPDialer is the default.
	Dialer Dialer

	// backoffRandomizationFactor is used in tests only: default randomization factor is used in produnction.
	// See https://godoc.org/github.com/cenkalti/backoff#ExponentialBackOff for more info
	backoffRandomizationFactor *float64
}

// FlagsParser is needed to embed goconnpool into production application.
// For example flag.FlagSet could be used here (https://golang.org/pkg/flag/#FlagSet) or you could implement
// your own config parser.
type FlagsParser interface {
	IntVar(dst *int, name string, def int, descr string)
	DurationVar(p *time.Duration, name string, value time.Duration, usage string)
}

// NewConfig initializes configuration using FlagsParser.
// If you use flag.FlagSet-based parsers, config will be filled only after Parse() method invoked.
func NewConfig(p FlagsParser) *Config {
	var c Config

	p.IntVar(&c.MaxConnsPerServer, "max_conns_per_server", DefaultMaxConnsPerServer,
		"Maximum number of opened connections per server")
	p.IntVar(&c.MaxRPS, "max_rps", DefaultMaxRPS,
		"Maximim number of requests per one server per second")

	p.DurationVar(&c.ConnectTimeout, "connect_timeout", 0,
		"Maximum amount of time a dial will wait for a connect to complete")

	p.DurationVar(&c.InitialBackoffInterval, "init_backoff_interval", DefaultInitBackoffInterval,
		"Initial backoff interval to retry requests")
	p.DurationVar(&c.MaxBackoffInterval, "max_backoff_interval", DefaultMaxBackoffInterval,
		"Maximum backoff interval to retry requests")

	return &c
}

func (c Config) withDefaults() Config {
	if c.MaxConnsPerServer == 0 {
		c.MaxConnsPerServer = DefaultMaxConnsPerServer
	}

	if c.MaxRPS == 0 {
		c.MaxRPS = DefaultMaxRPS
	}

	if c.InitialBackoffInterval == 0 {
		c.InitialBackoffInterval = DefaultInitBackoffInterval
	}

	if c.MaxBackoffInterval == 0 {
		c.MaxBackoffInterval = DefaultMaxBackoffInterval
	}

	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = DefaultConnectTimeout
	}

	if c.Clock == nil {
		c.Clock = SystemClock{}
	}

	if c.Logger == nil {
		c.Logger = DummyLogger{}
	}

	if c.Dialer == nil {
		c.Dialer = &TCPDialer{}
	}

	return c
}
