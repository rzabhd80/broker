package helpers

import (
	"os"
	"time"
)

func CheckFileExists(filePath string) bool {
	if _, fileFound := os.Stat(filePath); fileFound == nil {
		return true
	}
	return false
}

func GenerateUniqueID() int {
	timestamp := int(time.Now().UnixNano())
	//r := rand.New(rand.NewSource(time.Now().UnixNano()))
	//randomPart := r.Int()
	uniqueID := timestamp + 1
	if uniqueID < 0 {
		uniqueID = -uniqueID
	}
	return uniqueID
}

func IsInitiator(val string) bool {
	return val == "true"
}
