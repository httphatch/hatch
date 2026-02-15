package cmd

import "embed"

var embeddedAssets embed.FS //nolint:unused // read by cmd/app.go (darwin-only)
var appIconData []byte      //nolint:unused // read by cmd/app.go (darwin-only)

// SetAssets stores the embedded frontend filesystem for use by the app command.
func SetAssets(assets embed.FS) {
	embeddedAssets = assets
}

// SetAppIcon stores the embedded application icon for use by the app command.
func SetAppIcon(icon []byte) {
	appIconData = icon
}
