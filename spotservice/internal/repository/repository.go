package repository

import (
	"context"

	"github.com/dim-pep/Market2/spotservice/internal/domain"
)

type MarketsRepo interface {
	AddMarket(context.Context, string, string) (uint64, error)
	DisableMarket(context.Context, uint64) (bool, error)
	ViewMarketsByRoles(context.Context, []string) ([]domain.Market, error)
	CheckHealth() error
	Close() error
}
