package statusbar

type LogLevel int

const (
	LevelInfo    LogLevel = iota // 0
	LevelWarning                 // 1
	LevelError                   // 2
	LevelFatal                   // 3
)

func (l LogLevel) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
