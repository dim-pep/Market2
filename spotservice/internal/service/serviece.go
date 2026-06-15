package service

import (
	"context"

	"github.com/dim-pep/Market2/spotservice/internal/cache"
	"github.com/dim-pep/Market2/spotservice/internal/domain"
	"github.com/dim-pep/Market2/spotservice/internal/errs"
	"github.com/dim-pep/Market2/spotservice/internal/interceptors"
	"github.com/dim-pep/Market2/spotservice/internal/repository"
	"go.uber.org/zap"
)

type MarketService struct {
	marketsRepo repository.MarketsRepo
	logger      *zap.Logger
	CacheRepo   cache.CacheRepo
}

func NewMarketService(repo repository.MarketsRepo, logger *zap.Logger, CacheRepo cache.CacheRepo) *MarketService {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MarketService{
		marketsRepo: repo,
		logger:      logger,
		CacheRepo:   CacheRepo,
	}
}

func (ms *MarketService) CloseDBConnection() error {
	if ms == nil || ms.marketsRepo == nil {
		return nil
	}

	return ms.marketsRepo.Close()
}

func (ms *MarketService) CloseCacheClient() error {
	if ms == nil || ms.CacheRepo == nil {
		return nil
	}

	return ms.CacheRepo.Close()
}

func (ms *MarketService) ViewMarkets(ctx context.Context, req *domain.ViewMarketsRequest) (*domain.ViewMarketsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req == nil {
		return nil, errs.ErrNilViewMarketsRequest
	}

	if ms == nil || ms.marketsRepo == nil {
		return nil, errs.ErrMarketRepositoryNotConfigured
	}

	requestID := interceptors.RequestIDFromContext(ctx)

	logger := ms.logger.With(
		zap.String("request_id", requestID),
		zap.String("operation", "view_markets"),
	)

	logger.Info(
		"view markets started",
		zap.Strings("user_roles", req.UserRoles),
	)

	markets, err := ms.marketsRepo.ViewMarketsByRoles(ctx, req.UserRoles)
	if err != nil {
		logger.Warn(
			"view markets failed",
			zap.String("repository_method", "ViewMarketsByRoles"),
			zap.Error(err),
		)

		return nil, err
	}

	marketIDs := make([]uint64, 0, len(markets))
	for _, market := range markets {
		marketIDs = append(marketIDs, market.ID)
	}

	logger.Info(
		"view markets succeeded",
		zap.Int("markets_count", len(markets)),
		zap.Uint64s("market_ids", marketIDs),
	)

	return &domain.ViewMarketsResponse{
		Markets: markets,
	}, nil
}
