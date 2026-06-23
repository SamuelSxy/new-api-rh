package system_setting

import (
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type ThemeSettings struct {
	Frontend string `json:"frontend"`
}

var themeSettings = ThemeSettings{
	Frontend: "default",
}

func init() {
	config.GlobalConfig.Register("theme", &themeSettings)
	syncThemeToCommon()
}

func syncThemeToCommon() {
	if themeSettings.Frontend != "default" && themeSettings.Frontend != "classic" && themeSettings.Frontend != "hai" {
		themeSettings.Frontend = "default"
	}
	// The embedded frontend is fixed at build time. When THEME is set, the
	// build-pinned theme wins and the DB option must not override it.
	if os.Getenv("THEME") != "" {
		return
	}
	common.SetTheme(themeSettings.Frontend)
}

func GetThemeSettings() *ThemeSettings {
	return &themeSettings
}

// UpdateAndSyncTheme syncs the theme config to common after DB load.
func UpdateAndSyncTheme() {
	syncThemeToCommon()
}
