package sync

import (
	"sort"

	"deviceshadow/internal/model"
)

// ReplayPlan merges the offline cache replay and the online watermark resume
// into a single list without delivering any version twice.
type ReplayPlan struct {
	ops  []model.DesiredOp
	seen map[int64]struct{}
	max  int64
}

// NewReplayPlan builds a deduplicated plan from the cached and online ops.
// Cached ops win on equal versions because they were buffered first. The plan
// is emitted in ascending version order so the device always applies the
// newest desired state last; an old desired state can never land after, and
// therefore clobber, a newer push.
func NewReplayPlan(cached, online []model.DesiredOp) *ReplayPlan {
	plan := &ReplayPlan{seen: make(map[int64]struct{})}
	for _, op := range cached {
		plan.add(op)
	}
	for _, op := range online {
		plan.add(op)
	}
	// Sort by version so replay applies old state before new state. Without
	// this an out-of-order replay could push a stale desired state after the
	// latest one, rolling the device back.
	sort.Slice(plan.ops, func(i, j int) bool {
		return plan.ops[i].Version < plan.ops[j].Version
	})
	return plan
}

func (p *ReplayPlan) add(op model.DesiredOp) {
	if _, ok := p.seen[op.Version]; ok {
		return
	}
	p.seen[op.Version] = struct{}{}
	p.ops = append(p.ops, op)
	if op.Version > p.max {
		p.max = op.Version
	}
}

// Ops returns the delivery plan in ascending version order.
func (p *ReplayPlan) Ops() []model.DesiredOp {
	return p.ops
}

// Count returns how many ops the plan contains.
func (p *ReplayPlan) Count() int {
	return len(p.ops)
}
