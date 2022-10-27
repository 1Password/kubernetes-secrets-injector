package utils

import "strings"

func MakeBuildVersion(version string) string {
	parts := strings.Split(strings.ReplaceAll(version, "-beta", ""), ".")
	buildVersion := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) == 1 {
			buildVersion += "0" + parts[i]
		} else {
			buildVersion += parts[i]
		}
	}
	if len(parts) != 3 {
		return buildVersion
	}
	return buildVersion + "01"
}
