package planner

import (
	"context"
	"fmt"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

var ErrPlannerEntryNotFound = fmt.Errorf("planner entry not found")

type PlannerService struct {
	repo PlannerRepository
}

func NewPlannerService(repo PlannerRepository) *PlannerService {
	return &PlannerService{repo: repo}
}

func (s *PlannerService) CreatePlannerEntry(ctx context.Context, title string, days []string, startTime, endTime, color string) (*PlannerEntry, error) {
	id := shared.NewUUID()

	entry, err := NewPlannerEntry(id, title, days, startTime, endTime, color)
	if err != nil {
		return nil, fmt.Errorf("create planner entry: %w", err)
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("persist planner entry: %w", err)
	}

	log.Info().Str("entry_id", entry.ID).Str("title", entry.Title).Msg("planner entry created")
	return entry, nil
}

func (s *PlannerService) GetPlannerEntry(ctx context.Context, id string) (*PlannerEntry, error) {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *PlannerService) ListPlannerEntries(ctx context.Context) ([]*PlannerEntry, error) {
	return s.repo.List(ctx)
}

func (s *PlannerService) UpdatePlannerEntry(ctx context.Context, id string, title *string, days *[]string, startTime, endTime, color *string) (*PlannerEntry, error) {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := entry.Update(title, days, startTime, endTime, color); err != nil {
		return nil, fmt.Errorf("update planner entry: %w", err)
	}

	if err := s.repo.Update(ctx, entry); err != nil {
		return nil, fmt.Errorf("persist planner entry update: %w", err)
	}

	log.Info().Str("entry_id", entry.ID).Msg("planner entry updated")
	return entry, nil
}

func (s *PlannerService) DeletePlannerEntry(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	log.Info().Str("entry_id", id).Msg("planner entry deleted")
	return nil
}
