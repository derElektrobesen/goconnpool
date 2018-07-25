package goconnpool

import (
	context "context"
	net "net"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testServer struct {
	t   *testing.T
	ass *require.Assertions

	s   *server
	cfg Config

	dialerMock *Mockdialer
	clockMock  *clock.Mock
}

func newTestServer() testServer {
	return testServer{}
}

func (s testServer) withConfig(cfg Config) testServer {
	s.cfg = cfg
	return s
}

func (s testServer) wrap(cb func(s testServer)) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		t.Helper()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		s.t = t
		s.ass = require.New(t)

		s.dialerMock = NewMockdialer(ctrl)
		s.clockMock = clock.NewMock()
		s.clockMock.Set(time.Unix(1514764800, 0).UTC())

		s.cfg.Clock = s.clockMock

		s.s = newServer("tcp", "addr", s.cfg, s.dialerMock)

		cb(s)
	}
}

func testRatelimits(s testServer) {
	s.dialerMock.EXPECT().
		DialContext(gomock.Any(), gomock.Any(), gomock.Any()).
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
	s.ass.Error(err)

	// call #4, still ratelimited
	s.clockMock.Add(time.Millisecond)
	_, err = s.s.getConnection(context.Background())
	s.ass.Error(err)

	// call #5, still ratelimited
	s.clockMock.Add(98 * time.Millisecond)
	_, err = s.s.getConnection(context.Background())
	s.ass.Error(err)

	// call #6, timeout came
	s.clockMock.Add(2 * time.Millisecond)
	cn, err = s.s.getConnection(context.Background())
	s.ass.NotNil(cn)
	s.ass.NoError(err)
}

func testTooManyConns(s testServer) {
	s.dialerMock.EXPECT().
		DialContext(gomock.Any(), gomock.Any(), gomock.Any()).
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
}

func TestServer(t *testing.T) {
	t.Parallel()

	t.Run("ratelimits",
		newTestServer().
			withConfig(Config{
				MaxRPS:            10,
				MaxConnsPerServer: 10,
			}).
			wrap(testRatelimits),
	)

	t.Run("too_many_conns",
		newTestServer().
			withConfig(Config{
				MaxRPS:            99999999,
				MaxConnsPerServer: 3,
			}).
			wrap(testTooManyConns),
	)
}
