package goconnpool

import (
	context "context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func testConfigParsing(t *testing.T) {
	t.Parallel()

	s := flag.NewFlagSet("test", flag.PanicOnError)
	cfgPtr := NewConfig(s)
	require.NoError(t, s.Parse(
		strings.Split(
			"-max_conns_per_server 10 "+
				"-max_rps 20 "+
				"-connect_timeout 25ms "+
				"-init_backoff_interval 18s "+
				"-max_backoff_interval 46m ",
			" ",
		),
	))

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
			Dialer:                 &TCPDialer{},
		}, s.cfg)

	// just to increment code coverage: nothing to test
	s.cfg.Logger.Printf(">>>")
	s.cfg.Clock.Now()
	s.cfg.Clock.Since(time.Now())
	s.cfg.Clock.After(0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // to prevent locking

	s.cfg.Dialer.Dial(ctx, "127.0.0.1:95328") // nolint:errcheck
}

func testDefaultConnPoolCreation(t *testing.T) {
	// this test is also exisists just to increment code coverage: nothing to test here %)
	t.Parallel()

	s := NewConnPool(Config{})
	s.RegisterServer("y")
}

func newTestConnProviderFactory(srvs ...connectionProvider) func(addr string, cfg Config) connectionProvider {
	return func(addr string, cfg Config) connectionProvider {
		if len(srvs) == 0 {
			panic("unexpected call of conn provider factory")
		}

		srv := srvs[0]
		srvs = srvs[1:]
		return srv
	}
}

type testLogger struct {
	t *testing.T
}

func (l testLogger) Printf(format string, args ...interface{}) {
	l.t.Helper()
	l.t.Logf(format, args...)
}

func testOpenConnNonBlock(t *testing.T) {
	t.Parallel()
	ass := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	p := newConnPool(Config{
		Logger: testLogger{t: t},
	})
	srv1 := NewMockconnectionProvider(ctrl)
	srv2 := NewMockconnectionProvider(ctrl)
	srv3 := NewMockconnectionProvider(ctrl)
	p.connProviderFactory = newTestConnProviderFactory(srv1, srv2, srv3)

	_, err := p.OpenConnNonBlock(context.Background())
	ass.Equal(ErrNoServersRegistered, err)

	p.RegisterServer("y") // srv1
	p.RegisterServer("k") // srv2
	p.RegisterServer("m") // srv3

	srv1.EXPECT().retryTimeout().AnyTimes()
	srv2.EXPECT().retryTimeout().AnyTimes()
	srv3.EXPECT().retryTimeout().AnyTimes()

	cn := &serverConn{}
	srv1.EXPECT().getConnection(gomock.Any()).Return(cn, nil)
	gotCn, err := p.OpenConnNonBlock(context.Background())
	ass.NoError(err)
	ass.Equal(cn, gotCn)

	cn = &serverConn{}
	srv2.EXPECT().getConnection(gomock.Any()).Return(cn, nil)
	gotCn, err = p.OpenConnNonBlock(context.Background())
	ass.NoError(err)
	ass.Equal(cn, gotCn)

	cn = &serverConn{}
	gomock.InOrder(
		srv3.EXPECT().getConnection(gomock.Any()).Return(nil, errRatelimited),
		srv1.EXPECT().getConnection(gomock.Any()).Return(cn, nil),
	)

	gotCn, err = p.OpenConnNonBlock(context.Background())
	ass.NoError(err)
	ass.Equal(cn, gotCn)

	gomock.InOrder(
		srv2.EXPECT().getConnection(gomock.Any()).Return(nil, errRatelimited),
		srv3.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
		srv1.EXPECT().getConnection(gomock.Any()).Return(nil, fmt.Errorf("xxx")),
	)

	_, err = p.OpenConnNonBlock(context.Background())
	ass.Error(err)
	ass.Equal("some servers are down, other ratelimited", err.Error())

	gomock.InOrder(
		srv2.EXPECT().getConnection(gomock.Any()).Return(nil, errRatelimited),
		srv3.EXPECT().getConnection(gomock.Any()).Return(nil, errRatelimited),
		srv1.EXPECT().getConnection(gomock.Any()).Return(nil, errRatelimited),
	)

	_, err = p.OpenConnNonBlock(context.Background())
	ass.Error(err)
	ass.Equal("all servers are ratelimited", err.Error())

	gomock.InOrder(
		srv2.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
		srv3.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
		srv1.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
	)

	_, err = p.OpenConnNonBlock(context.Background())
	ass.Error(err)
	ass.Equal("all servers are down", err.Error())
}

func testOpenConnBlock(t *testing.T) {
	t.Parallel()

	ass := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cl := clock.NewMock()

	p := newConnPool(Config{
		Logger: testLogger{t: t},
		Clock:  cl,
	})
	srv1 := NewMockconnectionProvider(ctrl)
	srv2 := NewMockconnectionProvider(ctrl)
	p.connProviderFactory = newTestConnProviderFactory(srv1, srv2)

	// check call will not be blocked if no servers were passed
	_, err := p.OpenConn(context.Background())
	ass.Equal(ErrNoServersRegistered, err)

	p.RegisterServer("y")
	p.RegisterServer("yt")

	// check success connection opening
	cn := &serverConn{}
	srv1.EXPECT().getConnection(gomock.Any()).Return(cn, nil)
	gotCn, err := p.OpenConn(context.Background())
	ass.NoError(err)
	ass.Equal(cn, gotCn)

	// check retry timeout
	cn = &serverConn{}
	gomock.InOrder(
		srv2.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
		srv1.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
		srv2.EXPECT().getConnection(gomock.Any()).Return(cn, nil),
	)

	gomock.InOrder(
		srv2.EXPECT().retryTimeout().Return(time.Minute),
		srv1.EXPECT().retryTimeout().Return(time.Second),
	)

	ready := make(chan struct{})
	go func() {
		// gomock could be much easier to use in this case, but it not supports chans =(
		go func() {
			time.Sleep(100 * time.Millisecond)
			ready <- struct{}{}
		}()
		gotCn, err = p.OpenConn(context.Background())
		close(ready)
	}()

	<-ready
	cl.Add(time.Minute)
	<-ready

	ass.NoError(err)
	ass.Equal(cn, gotCn)

	// check loop could be broken with context
	ctx, cancel := context.WithCancel(context.Background())

	gomock.InOrder(
		srv1.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
		srv2.EXPECT().getConnection(gomock.Any()).Return(nil, errServerIsDown),
	)

	gomock.InOrder(
		srv1.EXPECT().retryTimeout().Return(time.Second),
		srv2.EXPECT().retryTimeout().Return(time.Minute),
	)

	ready = make(chan struct{})
	go func() {
		ready <- struct{}{}
		_, err = p.OpenConn(ctx)
		close(ready)
	}()

	<-ready
	cancel()
	<-ready

	ass.Error(err)
	ass.Equal("operation cancelled", err.Error())
}

func testOpenConn(t *testing.T) {
	t.Parallel()

	t.Run("open_conn_non_block", testOpenConnNonBlock)
	t.Run("open_conn_block", testOpenConnBlock)
}

func testOpenConnection(t *testing.T) {
	t.Parallel()

	t.Run("create_default_conn_pool", testDefaultConnPoolCreation)
	t.Run("open_conn", testOpenConn)
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
