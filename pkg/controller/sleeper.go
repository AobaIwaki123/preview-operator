package controller

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/AobaIwaki123/preview-operator/pkg/k8s"
)

// Sleeper tracks access time and scales down deployments after idle timeout.
type Sleeper struct {
	scaler      *k8s.Scaler
	idleTimeout time.Duration
	checkPeriod time.Duration

	mu          sync.RWMutex
	lastAccess  time.Time
}

// NewSleeper creates a new Sleeper instance.
func NewSleeper(scaler *k8s.Scaler, idleTimeout time.Duration) *Sleeper {
	return &Sleeper{
		scaler:      scaler,
		idleTimeout: idleTimeout,
		checkPeriod: 30 * time.Second,
		lastAccess:  time.Now(),
	}
}

// RecordAccess updates the last access timestamp to now.
func (s *Sleeper) RecordAccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAccess = time.Now()
}

// GetLastAccess returns the current last access timestamp.
func (s *Sleeper) GetLastAccess() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAccess
}

// Start runs the idle check loop in background until context is cancelled.
func (s *Sleeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.checkPeriod)
	defer ticker.Stop()

	log.Printf("[Sleeper] Started idle watcher (timeout: %v, check period: %v)", s.idleTimeout, s.checkPeriod)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Sleeper] Stopping idle watcher.")
			return
		case <-ticker.C:
			s.checkAndScaleDown(ctx)
		}
	}
}

func (s *Sleeper) checkAndScaleDown(parentCtx context.Context) {
	idleDuration := time.Since(s.GetLastAccess())
	if idleDuration < s.idleTimeout {
		return
	}

	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()

	replicas, err := s.scaler.GetReplicas(ctx)
	if err != nil {
		log.Printf("[Sleeper] Error getting replicas: %v", err)
		return
	}

	if replicas > 0 {
		log.Printf("[Sleeper] Idle timeout reached (%v > %v). Scaling down to 0 replicas", idleDuration, s.idleTimeout)
		if err := s.scaler.ScaleTo(ctx, 0); err != nil {
			log.Printf("[Sleeper] Failed to scale down: %v", err)
		}
	}
}
