package quota

import (
	"math"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
)

const (
	limitFree int64 = 128 * 1024 * 1024
	limitPro  int64 = 256 * 1024 * 1024
	limitMax  int64 = 512 * 1024 * 1024
)

var storagePlanLimits = map[string]int64{
	domain.PlanFree:   limitFree,
	domain.PlanFreeze: limitFree,
	domain.PlanPro:    limitPro,
	domain.PlanMax:    limitMax,
	domain.PlanAdmin:  math.MaxInt64,
}

func LimitFor(plan string) int64 {
	if limit, ok := storagePlanLimits[plan]; ok {
		return limit
	}
	return limitFree
}

func IsUnlimited(plan string) bool {
	return plan == domain.PlanAdmin
}
