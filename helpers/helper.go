package helpers

import (
	"os"
)

func CheckFileExists(filePath string) bool {
	if _, fileFound := os.Stat(filePath); fileFound == nil {
		return true
	}
	return false
}
