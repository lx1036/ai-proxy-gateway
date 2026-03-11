package util

import "os"

func LookupEnv(key, defaultVal string) string {
	result, ok := os.LookupEnv(key)
	if !ok {
		result = defaultVal
	}

	return result
}
