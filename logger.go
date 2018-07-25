package goconnpool

// Logger interface is used to log some messages into user log.
type Logger interface {
	Printf(format string, args ...interface{})
}

// DummyLogger is used to skip any message printed by the package.
type DummyLogger struct{}

func (DummyLogger) Printf(format string, args ...interface{}) {}
