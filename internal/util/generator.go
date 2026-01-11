package util

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func GenerateShortCode(secret string, id int64) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", secret, id)))
	return base64.RawURLEncoding.EncodeToString(h[:6])
}
