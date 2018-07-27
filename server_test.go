package goconnpool

import (
	context "context"
	"fmt"
	"math"
	net "net"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type testServer struct {
	t   *testing.T
	ass *require.Assertions

	s   *server
	cfg Config

	ctrl *gomock.Controller

	dialerMock *MockDialer
	clockMock  *clock.Mock
}

func newTestServer() testServer {
	return testServer{}
}

func (s testServer) withConfig(cfg Config) testServer {
	var empty Config
	if s.cfg != empty {
		panic("call withConfig() before all modifications")
	}

	s.cfg = cfg
	return s
}

func (s testServer) withoutTimeouts() testServer {
	s.cfg.ConnectTimeout = 24 * time.Hour
	return s
}

func (s testServer) withoutRateLimits() testServer {
	s.cfg.MaxRPS = math.MaxInt32
	s.cfg.MaxConnsPerServer = math.MaxInt32
	return s
}

func (s testServer) getConnection() (Conn, error) {
	s.t.Helper()

	s.clockMock.Add(time.Second)
	return s.s.getConnection(context.Background())
}

func (s testServer) getConnectionNoError() Conn {
	s.t.Helper()
	cn, err := s.getConnection()
	s.ass.NoError(err)
	return cn
}

func (s testServer) wrap(cb func(s testServer)) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		t.Helper()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		s.t = t
		s.ass = require.New(t)

		s.dialerMock = NewMockDialer(ctrl)
		s.clockMock = clock.NewMock()
		s.clockMock.Set(time.Unix(1514764800, 0).UTC())

		s.cfg.Clock = s.clockMock
		s.cfg.Dialer = s.dialerMock
		s.ctrl = ctrl

		s.s = newServer("addr", s.cfg)

		cb(s)
	}
}

func testRatelimits(s testServer) {
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(&net.IPConn{}, nil).
		Times(3)

	// call #1, always not ratelimited
	cn, err := s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)

	s.clockMock.Add(time.Second) // 10 requests per second is expected here (1 request per 100 ms)

	// call #2, don't limit this request
	cn, err = s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)

	// call #3, ratelimited
	_, err = s.s.getConnection(context.Background())
	s.ass.Equal(errRatelimit, errors.Cause(err))

	s.ass.Equal(100*time.Millisecond, s.s.retryTimeout())

	// call #4, still ratelimited
	s.clockMock.Add(time.Millisecond)
	_, err = s.s.getConnection(context.Background())
	s.ass.Equal(errRatelimit, errors.Cause(err))

	// call #5, still ratelimited
	s.clockMock.Add(98 * time.Millisecond)
	_, err = s.s.getConnection(context.Background())
	s.ass.Equal(errRatelimit, errors.Cause(err))

	// call #6, timeout came
	s.clockMock.Add(2 * time.Millisecond)
	cn, err = s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)
}

func testTooManyConns(s testServer) {
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(&net.IPConn{}, nil).
		Times(3)

	cn, err := s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)

	s.clockMock.Add(time.Second)
	cn, err = s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)

	s.clockMock.Add(time.Second)
	cn, err = s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)

	s.clockMock.Add(time.Second)
	_, err = s.s.getConnection(context.Background())
	s.ass.Error(err)
	s.ass.Equal(errRatelimit, errors.Cause(err))

	s.ass.Equal(100*time.Millisecond, s.s.retryTimeout())
}

type closer interface {
	Close()
}

type closableTestConn struct {
	net.Conn
	closer closer
	err    error
}

func (cn *closableTestConn) Close() error {
	cn.closer.Close() // to be sure connection were really closed
	return cn.err
}

func (s testServer) newClosableTestConn(err error, needClose bool) net.Conn {
	cl := NewMockcloser(s.ctrl)

	if needClose {
		cl.EXPECT().Close()
	}

	return &closableTestConn{
		Conn:   &net.IPConn{},
		err:    err,
		closer: cl,
	}
}

func (s testServer) newClosableTestConnFactory(
	err error,
	needClose bool,
) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return s.newClosableTestConn(nil, needClose), err
	}
}

func testConnsReuse(s testServer) {
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		DoAndReturn(s.newClosableTestConnFactory(nil, false)).
		Times(2)

	cn1 := s.getConnectionNoError()
	s.ass.NoError(cn1.ReturnToPool())

	cn := s.getConnectionNoError()
	s.ass.Equal(cn1.OriginalConn(), cn.OriginalConn())

	cn2 := s.getConnectionNoError()
	s.ass.NoError(cn2.ReturnToPool())

	cn = s.getConnectionNoError()
	s.ass.Equal(cn1.OriginalConn(), cn.OriginalConn())

	s.ass.NoError(cn.ReturnToPool())

	cn = s.getConnectionNoError()
	s.ass.Equal(cn2.OriginalConn(), cn.OriginalConn())
}

func testBrokenConns(s testServer) {
	cn := s.newClosableTestConn(fmt.Errorf("xxx"), true)
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(cn, nil)

	cn1 := s.getConnectionNoError() // this connection is really wrapped `cn` connection
	s.ass.NoError(cn1.ReturnToPool())

	cn1 = s.getConnectionNoError() // reuse already opened connection
	s.ass.Error(cn1.Close())       // error equals to "xxx" here. Close() call is expected here

	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(&net.IPConn{}, nil).
		Times(3) // check number of opened conns was decremented

	for i := 0; i < 3; i++ {
		s.getConnectionNoError()
	}

	_, err := s.getConnection()
	s.ass.Error(err) // too many opened connections (see config)
}

func testServerIsDown(s testServer) {
	// Initial backoff interval == 1 min here
	// Max backoff interval == 5 mins here
	//
	// Backoff algorithm will generate timeouts in the following order:
	//  * 1m0s
	//  * 1m30s
	//  * 2m15s
	//  * 3m22.5s
	//  * 5m0s
	//  * 5m0s
	//
	// This order is guaranteed by zero randomization factor (see config initialization)

	ctx := context.Background()

	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("xxx"))

	_, err := s.s.getConnection(ctx) // Dial() returned an error here
	s.ass.Equal(errServerIsDown, errors.Cause(err))

	s.clockMock.Add(30 * time.Second) // 30s elapsed, nextBackoff == 1m

	_, err = s.s.getConnection(ctx) // Dial() shouldn't be called here: backoff interval wasn't passed
	s.ass.Equal(errServerIsDown, errors.Cause(err))

	s.ass.Equal(30*time.Second, s.s.retryTimeout())

	s.clockMock.Add(31 * time.Second) // 1m1s elapsed, nextBackoff == 1m

	s.ass.Equal(time.Duration(0), s.s.retryTimeout())

	// Test timeouts
	ready := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (net.Conn, error) { // nolint:unparam
			cancel()
			time.Sleep(10 * time.Millisecond)
			close(ready)
			return &net.TCPConn{}, nil
		})

	// backoff interval passed: retry connection (with error)
	// nextBackoff == 1m1s + 1m30s == 2m31s
	_, err = s.s.getConnection(ctx)
	s.ass.Equal(errServerIsDown, errors.Cause(err))

	<-ready // to be sure gomock returned control
	ctx = context.Background()

	// Check backoff interval updated: Dial() shouldn't be called here
	s.clockMock.Add(time.Minute) // 2m1s elapsed, nextBackoff == 2m31s
	_, err = s.s.getConnection(ctx)
	s.ass.Equal(errServerIsDown, errors.Cause(err))

	// Next Dial() will return correct connection
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(&net.IPConn{}, nil)

	s.clockMock.Add(31 * time.Second) // 2m32s elapsed, nextBackoff == 2m32s
	gotCn, err := s.s.getConnection(ctx)
	s.ass.NoError(err)

	_, ok := gotCn.OriginalConn().(*net.IPConn)
	s.ass.True(ok)

	// Check backoff correctly updated when server was up
	s.clockMock.Add(time.Microsecond) // required to not hit ratelimits
	s.dialerMock.EXPECT().
		Dial(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("zzz"))

	_, err = s.s.getConnection(ctx)
	s.ass.Equal(errServerIsDown, errors.Cause(err))
}

func testConnectionDoubleClose(s testServer) {
	gomock.InOrder(
		//cn1
		s.dialerMock.EXPECT().
			Dial(gomock.Any(), gomock.Any()).
			DoAndReturn(s.newClosableTestConnFactory(nil, true)),

		//cn2
		s.dialerMock.EXPECT().
			Dial(gomock.Any(), gomock.Any()).
			DoAndReturn(s.newClosableTestConnFactory(nil, false)),
	)

	cn1 := s.getConnectionNoError()
	s.ass.NoError(cn1.Close())
	s.ass.Error(cn1.ReturnToPool())
	s.ass.Error(cn1.Close())

	cn2 := s.getConnectionNoError()
	s.ass.NoError(cn2.ReturnToPool())
	s.ass.Error(cn2.Close())
	s.ass.Error(cn2.ReturnToPool())
}

func TestServer(t *testing.T) {
	t.Parallel()

	t.Run("ratelimits",
		newTestServer().
			withConfig(Config{
				MaxRPS:            10,
				MaxConnsPerServer: 10,
			}).
			withoutTimeouts().
			wrap(testRatelimits),
	)

	t.Run("too_many_conns",
		newTestServer().
			withConfig(Config{
				MaxRPS:            math.MaxInt32,
				MaxConnsPerServer: 3,
			}).
			withoutTimeouts().
			wrap(testTooManyConns),
	)

	t.Run("reuse_already_opened_conn",
		newTestServer().
			withoutRateLimits().
			withoutTimeouts().
			wrap(testConnsReuse),
	)

	t.Run("broken_connection",
		newTestServer().
			withConfig(Config{
				MaxRPS:            math.MaxInt32,
				MaxConnsPerServer: 3,
			}).
			withoutTimeouts().
			wrap(testBrokenConns),
	)

	var backoffRandomizationFactor float64
	t.Run("server_is_down",
		newTestServer().
			withConfig(Config{
				InitialBackoffInterval:     time.Minute,
				MaxBackoffInterval:         5 * time.Minute,
				backoffRandomizationFactor: &backoffRandomizationFactor,
			}).
			withoutRateLimits().
			withoutTimeouts().
			wrap(testServerIsDown),
	)

	t.Run("connection_double_close",
		newTestServer().
			withoutRateLimits().
			withoutTimeouts().
			wrap(testConnectionDoubleClose),
	)
}
