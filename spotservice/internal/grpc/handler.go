package handler

import (
	"context"
	"strings"

	"github.com/dim-pep/Market2/proto/pb/spot_service"
	"github.com/dim-pep/Market2/spotservice/internal/domain"
	"github.com/dim-pep/Market2/spotservice/internal/errs"
	"github.com/dim-pep/Market2/spotservice/internal/mapper"
	observability "github.com/dim-pep/Market2/spotservice/internal/metrics"
	"github.com/dim-pep/Market2/spotservice/internal/service"
	"github.com/dim-pep/Market2/spotservice/internal/validator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SportServerHandler struct {
	spot_service.UnimplementedSportInstrumentServiceServer

	marketService *service.MarketService
}

func NewSportServerHandler(marketService *service.MarketService) *SportServerHandler {
	return &SportServerHandler{
		marketService: marketService,
	}
}

func (h *SportServerHandler) Register(grpcServer grpc.ServiceRegistrar) {
	spot_service.RegisterSportInstrumentServiceServer(grpcServer, h)
}

func (h *SportServerHandler) ViewMarkets(ctx context.Context, req *spot_service.ViewMarketsRequest) (*spot_service.ViewMarketsResponse, error) {
	const (
		serviceName = "spot_service"
		methodName  = "view_markets"
	)

	defer observability.ObserveOperationDuration("spot_service", "view_markets")()

	if req == nil {
		observability.IncOperation(serviceName, methodName, codes.InvalidArgument.String())
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if h == nil || h.marketService == nil {
		observability.IncOperation(serviceName, methodName, codes.Internal.String())
		return nil, status.Error(codes.Internal, "market service is not configured")
	}

	validationErrors, ok := validator.NewValidator().
		ValidateViewMarkets(req).
		Validate()

	if !ok {
		observability.IncOperation(serviceName, methodName, codes.InvalidArgument.String())

		return nil, status.Errorf(
			codes.InvalidArgument,
			"validation failed: %s",
			strings.Join(validationErrors, "; "),
		)
	}

	domainReq := domain.ViewMarketsRequest{
		UserRoles: req.UserRoles,
	}

	domainResp, err := h.marketService.ViewMarkets(ctx, &domainReq)
	if err != nil {
		grpcErr := errs.DomainErrorToGrpcError(err)
		observability.IncOperation(serviceName, methodName, status.Code(grpcErr).String())

		return nil, grpcErr
	}

	protoMarkets := mapper.MarketsToProto(domainResp.Markets)

	observability.IncOperation(serviceName, methodName, codes.OK.String())

	return &spot_service.ViewMarketsResponse{
		Markets: protoMarkets,
	}, nil
}
