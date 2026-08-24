package sync

import "deviceshadow/internal/model"

// ReplayPlan merges the offline cache replay and the online watermark resume
// into a single list without delivering any version twice.
type ReplayPlan struct {
	ops  []model.DesiredOp
	seen map[int64]struct{}
	max  int64
}

// NewReplayPlan builds a deduplicated plan from the cached and online ops.
// Cached ops win on equal versions because they were buffered first.
func NewReplayPlan(cached, online []model.DesiredOp) *ReplayPlan {
	plan := &ReplayPlan{seen: make(map[int64]struct{})}
	for _, op := range cached {
		plan.add(op)
	}
	for _, op := range online {
		plan.add(op)
	}
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

// Ops returns the delivery plan in replay order.
func (p *ReplayPlan) Ops() []model.DesiredOp {
	return p.ops
}

// Count returns how many ops the plan contains.
func (p *ReplayPlan) Count() int {
	return len(p.ops)
}
