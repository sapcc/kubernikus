package util

import (
	"github.com/aokoli/goutils"
)

func GenerateBootstrapToken() string {
	token, _ := goutils.Random(16, 32, 127, true, true)
	return token
}
