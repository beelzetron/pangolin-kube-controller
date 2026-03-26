package controller

import (
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"sync"
	"sync/atomic"
	"time"

	logrus "github.com/sirupsen/logrus"
)

var jitterWarnOnce sync.Once

var fallbackJitterCounter atomic.Uint64

var cryptoRead = crand.Read

var jitterNow = time.Now
var jitterPID = os.Getpid

func (c *Controller) computeBackoffDuration(errorCount int) time.Duration {
	base := c.cfg.PollInterval
	if errorCount <= 0 {
		return base
	}
	d := base
	for i := 1; i < errorCount; i++ {
		if d >= c.cfg.MaxBackoff/2 {
			d = c.cfg.MaxBackoff
			break
		}
		d *= 2
	}
	if d > c.cfg.MaxBackoff {
		d = c.cfg.MaxBackoff
	}
	randFloat := getJitterFloat64()
	factor := 0.8 + randFloat*0.4
	return time.Duration(float64(d) * factor)
}

func getJitterFloat64() float64 {
	var b [8]byte
	if _, err := cryptoRead(b[:]); err == nil {
		randUint64 := binary.BigEndian.Uint64(b[:])
		const maxUint64PlusOne = float64(1<<63) * 2
		return float64(randUint64) / maxUint64PlusOne
	}
	jitterWarnOnce.Do(func() {
		logrus.Warnf("crypto/rand read failed; using fallback RNG for backoff jitter")
	})
	return fallbackJitterFloat64()
}

func fallbackJitterFloat64() float64 {
	seq := fallbackJitterCounter.Add(1)
	var seed [24]byte
	binary.BigEndian.PutUint64(seed[0:8], uint64(jitterNow().UnixNano()))
	binary.BigEndian.PutUint64(seed[8:16], uint64(jitterPID()))
	binary.BigEndian.PutUint64(seed[16:24], seq)
	sum := sha256.Sum256(seed[:])
	randUint64 := binary.BigEndian.Uint64(sum[0:8])
	const maxUint64PlusOne = float64(1<<63) * 2
	return float64(randUint64) / maxUint64PlusOne
}

func (c *Controller) sleepWithBackoff(ctx context.Context, errorCount int) {
	d := c.computeBackoffDuration(errorCount)
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
		return
	}
}
