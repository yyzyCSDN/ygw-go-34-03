package publish

import (
	"errors"

	"confighub/internal/model"
)

var ErrBadPercent = errors.New("gray percent must be between 0 and 100")

func ValidatePercent(p int) error {
	if p < 0 || p > 100 {
		return ErrBadPercent
	}
	return nil
}

func (g *Grader) List() []*model.GrayPlan {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*model.GrayPlan, 0, len(g.plans))
	for _, plan := range g.plans {
		out = append(out, plan)
	}
	return out
}

func (g *Grader) Remove(app model.AppID, group model.GroupID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.plans, model.NamespaceKey(app, group))
}
