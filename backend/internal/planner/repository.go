package planner

import "context"

type PlannerRepository interface {
	Create(ctx context.Context, entry *PlannerEntry) error
	GetByID(ctx context.Context, id string) (*PlannerEntry, error)
	List(ctx context.Context) ([]*PlannerEntry, error)
	Update(ctx context.Context, entry *PlannerEntry) error
	Delete(ctx context.Context, id string) error
}
