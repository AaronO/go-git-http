package github

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
)

// IsValidPayload checks if the github payload's hash fits with
// the hash computed by GitHub sent as a header
func IsValidPayload(headerHash string, payload []byte) {
	hash := HashPayload(payload)
	return hmac.Equal(
		[]byte(hash),
		[]byte(headerHash),
	)
}

// HashPayload computes the hash of payload's body according to the webhook's secret token
// see https://developer.github.com/webhooks/securing/#validating-payloads-from-github
// returning the hash as a hexadecimal string
func HashPayload(secret string, playloadBody []byte) string {
	hm := hmac.New(sha1.New, []byte(secret))
	sum := hm.Sum(playloadBody)
	return fmt.Sprintf("%x", sum)
}
