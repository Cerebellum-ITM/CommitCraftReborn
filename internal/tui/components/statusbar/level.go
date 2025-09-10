package statusbar

type LogLevel int

const (
	LevelInfo    LogLevel = iota // 0
	LevelWarning                 // 1
	LevelError                   // 2
	LevelFatal                   // 3
	LevelSuccess
)

func (l LogLevel) String() string {
	switch l {
	case LevelInfo:
		return "Info"
	case LevelWarning:
		return "Warning"
	case LevelError:
		return "Error"
	case LevelFatal:
		return "Fatal"
	case LevelSuccess:
		return "Success"
	default:
		return "UNKNOWN"
	}
}
