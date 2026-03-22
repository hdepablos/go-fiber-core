package processlifecyclemanager

import "context"

type CurrentVersionService interface {
	ResolveCurrentVersionID(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64, roadmap int) (int64, error)
}

type currentVersionService struct {
	resolveSvc ResolveService
}

func NewCurrentVersionService(resolveSvc ResolveService) CurrentVersionService {
	return &currentVersionService{resolveSvc: resolveSvc}
}

func (s *currentVersionService) ResolveCurrentVersionID(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64, roadmap int) (int64, error) {
	id, _, err := s.resolveSvc.ResolveProcessVersion(ctx, processTypeID, sedeID, overrideProcessVersionID, roadmap)
	return id, err
}

