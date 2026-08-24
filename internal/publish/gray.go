package publish

import (
	"hash/fnv"
	"strconv"
	"sync"

	"confighub/internal/model"
)

type Grader struct {
	mu    sync.RWMutex
	plans map[string]*model.GrayPlan
}

func NewGrader() *Grader {
	return &Grader{plans: make(map[string]*model.GrayPlan)}
}

func (g *Grader) Set(plan *model.GrayPlan) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.plans[model.NamespaceKey(plan.App, plan.Group)] = plan
}

func (g *Grader) Get(app model.AppID, group model.GroupID) (*model.GrayPlan, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	plan, ok := g.plans[model.NamespaceKey(app, group)]
	return plan, ok
}

// Evaluate reports whether a namespace is eligible to receive the gray
// revision. Eligibility is a pure function of the plan (percent hash and
// excluded groups) so that the rollout fraction is stable and independent of
// any client's history. A freshly connected client with no prior cursor must
// be judged by the same rule as everyone else; factoring the cursor in
// stranded new clients on the captured baseline forever.
func (g *Grader) Evaluate(plan *model.GrayPlan) bool {
	if plan == nil {
		return true
	}
	for _, group := range plan.ExcludedGroups {
		if group == plan.Group {
			return false
		}
	}
	if plan.Percent <= 0 {
		return false
	}
	if plan.Percent >= 100 {
		return true
	}
	h := fnv.New32a()
	h.Write([]byte(string(plan.Group)))
	h.Write([]byte(strconv.FormatInt(plan.CapturedRevision, 10)))
	return int(h.Sum32()%100) < plan.Percent
}
