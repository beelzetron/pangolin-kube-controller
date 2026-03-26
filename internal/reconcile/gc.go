package reconcile

import "time"

type GC struct {
	sem         chan struct{}
	delQueue    chan graceDeleteReq
	gracePeriod time.Duration
}

type graceDeleteReq struct {
}

func NewGC(workers int, gracePeriod time.Duration, queueSize int) *GC {
	if workers <= 0 {
		workers = 1
	}
	if queueSize <= 0 {
		queueSize = 256
	}
	return &GC{
		sem:         make(chan struct{}, workers),
		delQueue:    make(chan graceDeleteReq, queueSize),
		gracePeriod: gracePeriod,
	}
}

func (gc *GC) Workers() int {
	return cap(gc.sem)
}
