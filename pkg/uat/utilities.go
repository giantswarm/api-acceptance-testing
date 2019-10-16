package uat

import "strings"

func cleanupKeyPairID(id string) string {
	return strings.Replace(id, ":", "", -1)
}
