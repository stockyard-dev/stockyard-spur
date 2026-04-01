package server

import "github.com/stockyard-dev/stockyard-spur/internal/license"

type Limits struct {
	MaxProjects   int  // 0 = unlimited
	MaxEndpoints  int  // total
	RequestLog    bool // log incoming requests
	DelaySupport  bool // simulated latency
	RetentionDays int
}

var freeLimits = Limits{
	MaxProjects:   2,
	MaxEndpoints:  10,
	RequestLog:    true, // free hook
	DelaySupport:  false,
	RetentionDays: 7,
}

var proLimits = Limits{
	MaxProjects:   0,
	MaxEndpoints:  0,
	RequestLog:    true,
	DelaySupport:  true,
	RetentionDays: 90,
}

func LimitsFor(info *license.Info) Limits {
	if info != nil && info.IsPro() {
		return proLimits
	}
	return freeLimits
}

func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
