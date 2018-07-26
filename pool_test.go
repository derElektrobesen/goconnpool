package goconnpool

import (
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testConfigParsing(t *testing.T) {
	t.Parallel()

	s := flag.NewFlagSet("test", flag.PanicOnError)
	cfgPtr := NewConfig(s)
	s.Parse(
		strings.Split(
			"-max_conns_per_server 10 "+
				"-max_rps 20 "+
				"-connect_timeout 25ms "+
				"-init_backoff_interval 18s "+
				"-max_backoff_interval 46m ",
			" ",
		),
	)

	require.Equal(t,
		Config{
			MaxConnsPerServer:      10,
			MaxRPS:                 20,
			ConnectTimeout:         25 * time.Millisecond,
			InitialBackoffInterval: 18 * time.Second,
			MaxBackoffInterval:     46 * time.Minute,
		},
		*cfgPtr)
}

func testConfigFillDefaults(t *testing.T) {
	t.Parallel()

	s := newConnPool(Config{})
	require.Equal(t,
		Config{
			MaxConnsPerServer:      DefMaxConnsPerServer,
			MaxRPS:                 DefMaxRPS,
			ConnectTimeout:         DefConnectTimeout,
			InitialBackoffInterval: DefInitBackoffInterval,
			MaxBackoffInterval:     DefMaxBackoffInterval,
			Clock:                  SystemClock{},
			Logger:                 DummyLogger{},
		}, s.cfg)

	// just to increment code coverage
	s.cfg.Logger.Printf(">>>")
	s.cfg.Clock.Now()
	s.cfg.Clock.Since(time.Now())
}

func testOpenConnection(t *testing.T) {
	t.Parallel()
}

func testConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("config_parsing", testConfigParsing)
	t.Run("fill_defaults", testConfigFillDefaults)
}

func TestConnPool(t *testing.T) {
	t.Parallel()

	t.Run("config_defaults", testConfigDefaults)
	t.Run("open_connection", testOpenConnection)
}
