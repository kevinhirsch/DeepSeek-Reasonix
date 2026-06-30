package provider

import (
	"context"
	"sync"
	"time"
)

// KeyStatus tracks consecutive auth failures for a provider.
type KeyStatus struct {
	mu              sync.Mutex
	consecutive401s int
	last401         time.Time
	rotationPrompted bool
}

// KeyRotationDetector watches for consecutive 401 responses and triggers
// a key rotation prompt when 3+ consecutive requests return 401.
type KeyRotationDetector struct {
	mu       sync.Mutex
	statuses map[string]*KeyStatus
	onRotate func(providerName string) // called when rotation is needed
}

// NewKeyRotationDetector creates a key rotation detector.
func NewKeyRotationDetector(onRotate func(string)) *KeyRotationDetector {
	return &KeyRotationDetector{
		statuses: make(map[string]*KeyStatus),
		onRotate: onRotate,
	}
}

// RecordResult records an API response status for a provider.
// statusCode should be the HTTP status code from the provider response.
func (d *KeyRotationDetector) RecordResult(providerName string, statusCode int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ks := d.statuses[providerName]
	if ks == nil {
		ks = &KeyStatus{}
		d.statuses[providerName] = ks
	}
	if statusCode == 401 {
		ks.consecutive401s++
		ks.last401 = time.Now()
		if ks.consecutive401s >= 3 && !ks.rotationPrompted {
			ks.rotationPrompted = true
			if d.onRotate != nil {
				d.onRotate(providerName)
			}
		}
	} else if statusCode != 403 { // 403 is permissions, not auth
		ks.consecutive401s = 0
		ks.rotationPrompted = false
	}
}

// IsRotationNeeded reports whether key rotation has been triggered.
func (d *KeyRotationDetector) IsRotationNeeded(providerName string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	ks := d.statuses[providerName]
	return ks != nil && ks.rotationPrompted
}

// Reset clears the rotation flag for a provider (after key is updated).
func (d *KeyRotationDetector) Reset(providerName string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if ks := d.statuses[providerName]; ks != nil {
		ks.consecutive401s = 0
		ks.rotationPrompted = false
	}
}
