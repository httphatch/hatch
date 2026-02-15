//go:build !darwin

package tray

func setDockVisible(_ bool) {} //nolint:unused // darwin build stub

func setAppIcon(_ []byte) {} //nolint:unused // darwin build stub
