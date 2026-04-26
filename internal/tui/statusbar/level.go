package statusbar

type LogLevel int

// LogLevel values are picked by intent — never invent new ones.
//
//	LevelInfo    · neutral hints, ready / idle states
//	LevelSuccess · successful completion (rendered as OK)
//	LevelWarning · recoverable issue
//	LevelError   · failure that blocks user intent
//	LevelFatal   · unrecoverable error (rendered as ERROR)
//	LevelAI      · any AI/model activity
//	LevelRun     · long-running op in progress
//	LevelDebug   · verbose-only trace
const (
	LevelInfo LogLevel = iota
	LevelWarning
	LevelError
	LevelFatal
	LevelSuccess
	LevelAI
	LevelRun
	LevelDebug
)

// String returns the short uppercase label rendered inside the pill.
func (l LogLevel) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelSuccess:
		return "OK"
	case LevelWarning:
		return "WARN"
	case LevelError, LevelFatal:
		return "ERROR"
	case LevelAI:
		return "AI"
	case LevelRun:
		return "RUN"
	case LevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}
