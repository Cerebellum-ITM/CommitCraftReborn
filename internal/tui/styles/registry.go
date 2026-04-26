package styles

// themeFactory builds a fully-initialized theme. Stored in the registry so
// new themes can register themselves with one map entry.
type themeFactory func(useNerdFont bool) *Theme

var themeRegistry = map[string]themeFactory{
	"charmtone":    NewCharmtoneTheme,
	"harmonized":   NewHarmonizedTheme,
	"tokyonight":   NewTokyoNightTheme,
	"gruvbox-dark": NewGruvboxDarkTheme,
}

// DefaultThemeName is the fallback when the user hasn't configured one or
// asks for a name we don't know about.
const DefaultThemeName = "charmtone"

// GetTheme returns the theme registered under name. Falls back to the
// default theme when name is empty or unknown.
func GetTheme(name string, useNerdFont bool) *Theme {
	if factory, ok := themeRegistry[name]; ok {
		return factory(useNerdFont)
	}
	return themeRegistry[DefaultThemeName](useNerdFont)
}

// AvailableThemes returns the registered theme names so the UI can list
// them in a picker.
func AvailableThemes() []string {
	names := make([]string, 0, len(themeRegistry))
	for name := range themeRegistry {
		names = append(names, name)
	}
	return names
}
