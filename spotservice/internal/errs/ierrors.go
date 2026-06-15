package errs

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrCacheNotConfigured = errors.New("cache is not configured")
	ErrCacheMiss          = errors.New("cache entry not found")

	ErrMarketRepositoryNotConfigured = errors.New("market repository is not configured")
	ErrNilViewMarketsRequest         = errors.New("view markets request is nil")
	ErrAlreadyExist                  = errors.New("A market with these parameters already exists")

	ErrMarketServiceNotConfigured = errors.New("market service is not configured")
	ErrMarketNotFound             = errors.New("The requested market was not found")
	ErrMarketIsDeleted            = errors.New("The requested market is deleted")
	ErrNoAvailableMarkets         = errors.New("No markets are available right now, please try again later")
	ErrMarketAccessDenied         = errors.New("You don't have permission to access this market")

	ErrDatabaseConnection          = errors.New("Service temporarily unavailable, please try again later")
	ErrDatabaseUniqueViolation     = errors.New("An order with these parameters already exists")
	ErrDatabaseForeignKeyViolation = errors.New("Invalid reference to related resource")
	ErrOperationConflict           = errors.New("Operation conflict, please retry")

	ErrTimeout  = errors.New("Request timed out, please try again")
	ErrCanceled = errors.New("Request canceled, please try again")
	ErrInternal = errors.New("An internal server error occured, please try again later")
)

func DomainErrorToGrpcError(err error) error {
	if err == nil {
		return nil
	}

	// context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, ErrTimeout.Error())
	}
	if errors.Is(err, ErrCanceled) {
		return status.Error(codes.Canceled, ErrCanceled.Error())
	}

	// domain errors
	switch {
	case errors.Is(err, ErrAlreadyExist):
		return status.Error(codes.AlreadyExists, ErrAlreadyExist.Error())

	case errors.Is(err, ErrMarketNotFound):
		return status.Error(codes.NotFound, ErrMarketNotFound.Error())

	case errors.Is(err, ErrMarketIsDeleted):
		return status.Error(codes.NotFound, ErrMarketIsDeleted.Error())

	case errors.Is(err, ErrNoAvailableMarkets):
		return status.Error(codes.FailedPrecondition, ErrNoAvailableMarkets.Error())

	case errors.Is(err, ErrMarketAccessDenied):
		return status.Error(codes.PermissionDenied, ErrMarketAccessDenied.Error())
	}

	// db specific errors
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique violation
			return status.Error(codes.AlreadyExists, ErrDatabaseUniqueViolation.Error())
		case "23503": // foreign key violation
			return status.Error(codes.InvalidArgument, ErrDatabaseForeignKeyViolation.Error())
		case "40001": // serialization_failure
			return status.Error(codes.Aborted, ErrOperationConflict.Error())
		case "08006", "57P01", "57P02", "57P03", "08000", "08001", "08003": // connection issues
			return status.Error(codes.Unavailable, ErrDatabaseConnection.Error())
		case "57014":
			return status.Error(codes.DeadlineExceeded, ErrTimeout.Error())
		}
	}

	// infra errors
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, sql.ErrTxDone) {
		return status.Error(codes.Unavailable, ErrDatabaseConnection.Error())
	}

	return status.Error(codes.Internal, ErrInternal.Error())
}
