package goconnpool

// Logger interface is used to log some messages into user log.
type Logger interface {
	Errorf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

// DummyLogger is used to skip any message printed by the package.
type DummyLogger struct{}

func (DummyLogger) Errorf(format string, args ...interface{}) {}
func (DummyLogger) Infof(format string, args ...interface{})  {}
