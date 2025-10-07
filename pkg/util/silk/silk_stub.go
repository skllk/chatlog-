//go:build !cgo
// +build !cgo

package silk

import "fmt"

// Silk2MP3 returns an explicit error when chatlog is built without CGO support.
func Silk2MP3(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("silk to mp3 conversion requires cgo-enabled build")
}
