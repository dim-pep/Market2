package service

import (
	"context"

	"github.com/dim-pep/Market2/spotservice/interceptors"
	"github.com/dim-pep/Market2/spotservice/internal/cache"
	"github.com/dim-pep/Market2/spotservice/internal/domain"
	"github.com/dim-pep/Market2/spotservice/internal/repository"

	"go.uber.org/zap"
)

type MarketService struct {
	marketsRepo repository.MarketsRepo
	logger      *zap.Logger
	cacheClient cache.CacheRepo
}

func NewMarketService(repo repository.MarketsRepo, logger *zap.Logger, cache cache.CacheRepo) *MarketService {
	return &MarketService{marketsRepo: repo, logger: logger, cacheClient: cache}
}

func (ms *MarketService) CloseDBConnection() error {
	if ms.marketsRepo == nil {
		return nil
	}

	return ms.marketsRepo.Close()
}

func (os *MarketService) CloseCacheClient() error {
	if os.cacheClient == nil {
		return nil
	}

	return os.cacheClient.Close()
}

func (ms *MarketService) ViewMarkets(ctx context.Context, req *domain.ViewMarketsRequest) (*domain.ViewMarketsResponse, error) {
	rid := interceptors.XRequestFromContext(ctx)

	logger := ms.logger.With(
		zap.String("request_id", rid),
	)

	if err := ctx.Err(); err != nil {

		ms.logger.Warn("view_markets.failed", zap.Error(err))
		return nil, err
	}

	logger.Info("view_markets.started", zap.Strings("user_roles", req.UserRoles))

	markets, err := ms.marketsRepo.ViewMarketsByRoles(ctx, req.UserRoles)
	if err != nil {

		ms.logger.Warn("view_markets.failed",
			zap.String("operation", "markets_repo.view_markets_by_roles"),
			zap.Error(err),
		)
		return nil, err
	}

	marketIDs := make([]uint64, 0, len(markets))
	for _, m := range markets {
		marketIDs = append(marketIDs, m.ID)
	}

	logger.Info("view_markets.succeeded",
		zap.Int("markets_count", len(markets)),
		zap.Uint64s("market_ids", marketIDs),
	)

	return &domain.ViewMarketsResponse{Markets: markets}, nil
}
