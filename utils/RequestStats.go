package utils

import (
	"fmt"
	"sync/atomic"
)

type RespType int

const (
	PurchaseSuccess RespType = iota
	PurchaseFail
	ResponseFail
)

type RequestStats struct {
	TotalRequestCount    *atomic.Uint64
	PurchaseSuccessCount *atomic.Uint64
	PurchaseFailCount    *atomic.Uint64
	FailedRequestCount   *atomic.Uint64

	TotalNanoSeconds    *atomic.Uint64
	PurchaseSuccessNano *atomic.Uint64
	PurchaseFailNano    *atomic.Uint64
	FailedNanoSeconds   *atomic.Uint64
}

func NewRequestStats() *RequestStats {
	return &RequestStats{
		TotalRequestCount:    &atomic.Uint64{},
		PurchaseSuccessCount: &atomic.Uint64{},
		PurchaseFailCount:    &atomic.Uint64{},
		FailedRequestCount:   &atomic.Uint64{},
		TotalNanoSeconds:     &atomic.Uint64{},
		PurchaseSuccessNano:  &atomic.Uint64{},
		PurchaseFailNano:     &atomic.Uint64{},
		FailedNanoSeconds:    &atomic.Uint64{},
	}
}

func (s *RequestStats) Record(respType RespType, ns uint64) {
	s.TotalRequestCount.Add(1)
	s.TotalNanoSeconds.Add(ns)
	switch respType {
	case PurchaseSuccess:
		{
			s.PurchaseSuccessCount.Add(1)
			s.PurchaseSuccessNano.Add(ns)
		}
	case PurchaseFail:
		{
			s.PurchaseFailCount.Add(1)
			s.PurchaseFailNano.Add(ns)
		}
	case ResponseFail:
		{
			s.FailedRequestCount.Add(1)
			s.FailedNanoSeconds.Add(ns)
		}
	}
}

func (s *RequestStats) formatBlock(
	title string,
	count uint64,
	ns uint64,
) string {
	seconds := ns / 1e9
	var qps, avg float64
	if count == 0 {
		qps = 0.0
		avg = 0.0
	} else {
		qps = float64(count) / float64(seconds)
		avg = 1 * 1000 / qps
	}
	return fmt.Sprintf(
		"%s:\n  count: %d (%.2f qps)\n  Avg Duration(ms): %v\n",
		title, count, qps, avg,
	)
}

func (s *RequestStats) String() string {
	return s.formatBlock("total", s.TotalRequestCount.Load(), s.TotalNanoSeconds.Load()) +
		s.formatBlock("purchase success", s.PurchaseSuccessCount.Load(), s.PurchaseSuccessNano.Load()) +
		s.formatBlock("purchase failed", s.PurchaseFailCount.Load(), s.PurchaseFailNano.Load()) +
		s.formatBlock("response failed", s.FailedRequestCount.Load(), s.FailedNanoSeconds.Load())
}
