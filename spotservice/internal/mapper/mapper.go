package mapper

import (
	"github.com/dim-pep/Market2/spotservice/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MarketsToProto(markets []domain.Market) []*sport_service.Market {
	result := make([]*spot_service.Market, 0, len(markets))

	for _, market := range markets {
		result = append(result, MarketToProto(market))
	}

	return result
}

func MarketToProto(market domain.Market) *sport_service.Market {
	var deletedAt *timestamppb.Timestamp

	if market.DeletedAt != nil && !market.DeletedAt.IsZero() {
		deletedAt = timestamppb.New(*market.DeletedAt)
	}

	return &spot_service.Market{
		Id:        market.ID,
		Enabled:   market.Enabled,
		DeletedAt: deletedAt,
	}
}
