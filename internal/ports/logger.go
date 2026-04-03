package ports

type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}
