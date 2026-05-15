package quota_test

import (
	"math"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/quota"
)

func TestLimitFor(t *testing.T) {
	cases := []struct {
		plan string
		want int64
	}{
		{domain.PlanFree, 128 * 1024 * 1024},
		{domain.PlanFreeze, 128 * 1024 * 1024},
		{domain.PlanPro, 256 * 1024 * 1024},
		{domain.PlanMax, 512 * 1024 * 1024},
		{domain.PlanAdmin, math.MaxInt64},
		{"unknown", 128 * 1024 * 1024},
		{"", 128 * 1024 * 1024},
	}
	for _, c := range cases {
		if got := quota.LimitFor(c.plan); got != c.want {
			t.Errorf("LimitFor(%q) = %d, want %d", c.plan, got, c.want)
		}
	}
}

func TestIsUnlimited(t *testing.T) {
	if !quota.IsUnlimited(domain.PlanAdmin) {
		t.Error("admin must be unlimited")
	}
	if quota.IsUnlimited(domain.PlanFree) {
		t.Error("free must not be unlimited")
	}
}
