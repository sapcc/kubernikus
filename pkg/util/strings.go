package util

func DisabledValue(value string) bool {
	for _, s := range []string{"false", "False", "FALSE", "off", "Off", "OFF", "Disabled", "disabled", "no", "No", "NO"} {
		if value == s {
			return true
		}
	}
	return false
}

func EnabledValue(value string) bool {
	for _, s := range []string{"true", "True", "TRUE", "on", "On", "ON", "enabled", "Enabled", "yes", "Yes", "YES"} {
		if value == s {
			return true
		}
	}
	return false
}
