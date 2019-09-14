package internal

// NoLogger is a fake logger.
type NoLogger struct{}

// Debug emits a debug message.
func (NoLogger) Debug(msg string) {}

// Debugf formats and emits a debug message.
func (NoLogger) Debugf(format string, v ...interface{}) {}

// Info emits an informational message.
func (NoLogger) Info(msg string) {}

// Infof format and emits an informational message.
func (NoLogger) Infof(format string, v ...interface{}) {}

// Warn emits a warning message.
func (NoLogger) Warn(msg string) {}

// Warnf formats and emits a warning message.
func (NoLogger) Warnf(format string, v ...interface{}) {}
