package app

import "runtime/debug"

var buildVersion = "dev"

func currentVersion() string {
	if buildVersion != "" && buildVersion != "dev" {
		return buildVersion
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return buildVersion
	}
	return info.Main.Version
}
