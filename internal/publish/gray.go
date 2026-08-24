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

func (g *Grader) Evaluate(plan *model.GrayPlan, _ int64) bool {
	if plan == nil {
		return true
	}
	if isExcluded(plan) {
		return false
	}
	return withinPercent(plan)
}

func isExcluded(plan *model.GrayPlan) bool {
	for _, group := range plan.ExcludedGroups {
		if group == plan.Group {
			return true
		}
	}
	return false
}

func withinPercent(plan *model.GrayPlan) bool {
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
