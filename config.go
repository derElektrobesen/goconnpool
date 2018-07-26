package goconnpool

import "time"

const (
	// DefMaxConnsPerServer is the default value for MaxConnsPerServer config variable.
	DefMaxConnsPerServer = 1

	// DefMaxRPS is the default value for MaxRPS config variable.
	DefMaxRPS = 100

	// DefConnectTimeout is the default value for ConnectTimeout config variable.
	DefConnectTimeout = 5 * time.Second

	// DefInitBackoffInterval is the default value for InitialBackoffInterval config variable.
	DefInitBackoffInterval = 100 * time.Millisecond

	// DefMaxBackoffInterval is the default value for MaxBackoffInterval config variable.
	DefMaxBackoffInterval = 30 * time.Second
)

// Config holds some fields required during new connection establishing.
type Config struct {
	// MaxConnsPerServer declares maximum number of opened connections per one registered server.
	// Default value is DefMaxConnsPerServer.
	MaxConnsPerServer int

	// MaxRPS declares muximum number of requests which could be sent into one server (XXX: not connection)
	// per second.
	// Frankly this value regulates only a number of OpenConn calls: goconnpool can't regulate number of real
	// network requests.
	//
	// Default is DefMaxRPS.
	MaxRPS int

	// ConnectTimeout is the maximum amount of time a dial will wait for
	// a connect to complete.
	//
	// Default is DefConnectTimeout.
	ConnectTimeout time.Duration

	// InitialBackoffInterval configures InitialInterval for ExponentialBackOff algorithm.
	// See https://godoc.org/github.com/cenkalti/backoff#ExponentialBackOff for more info.
	//
	// Default is DefInitBackoffInterval
	InitialBackoffInterval time.Duration

	// MaxBackoffInterval configures MaxInterval for ExponentialBackOff algorithm.
	// See https://godoc.org/github.com/cenkalti/backoff#ExponentialBackOff for more info.
	//
	// Default is DefMaxBackoffInterval
	MaxBackoffInterval time.Duration

	// Clock could be used to reimplement behaviour of system clock.
	// SystemClock by default.
	Clock Clock

	// Logger could be used to view some messages, printed by the library.
	// This messages could contain information about server status, about some errors or something else.
	Logger Logger

	// Dialer is used to dial to the specific server.
	//
	// Required because some protocols (like thrift) can't be used with raw net.Conn object:
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

	p.IntVar(&c.MaxConnsPerServer, "max_conns_per_server", DefMaxConnsPerServer,
		"Maximum number of opened connections per server")
	p.IntVar(&c.MaxRPS, "max_rps", DefMaxRPS,
		"Maximim number of requests per one server per second")

	p.DurationVar(&c.ConnectTimeout, "connect_timeout", 0,
		"Maximum amount of time a dial will wait for a connect to complete")

	p.DurationVar(&c.InitialBackoffInterval, "init_backoff_interval", DefInitBackoffInterval,
		"Initial backoff interval to retry requests")
	p.DurationVar(&c.MaxBackoffInterval, "max_backoff_interval", DefMaxBackoffInterval,
		"Maximum backoff interval to retry requests")

	return &c
}

func (c Config) withDefaults() Config {
	if c.MaxConnsPerServer == 0 {
		c.MaxConnsPerServer = DefMaxConnsPerServer
	}

	if c.MaxRPS == 0 {
		c.MaxRPS = DefMaxRPS
	}

	if c.InitialBackoffInterval == 0 {
		c.InitialBackoffInterval = DefInitBackoffInterval
	}

	if c.MaxBackoffInterval == 0 {
		c.MaxBackoffInterval = DefMaxBackoffInterval
	}

	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = DefConnectTimeout
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
