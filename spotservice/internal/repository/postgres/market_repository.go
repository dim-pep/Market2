package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/dim-pep/Market2/spotservice/internal/domain"
	"github.com/dim-pep/Market2/spotservice/internal/errs"
	"github.com/lib/pq"
)

func (pr *postgresMarketsRepo) AddMarket(ctx context.Context, symbol string, allowedRoles string) (id uint64, err error) {

	query := `INSERT INTO markets (symbol,allowed_roles) VALUES ($1, $2) RETURNING id`

	tx, err := pr.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	rolesSlice := convertToSlice(allowedRoles)

	err = tx.QueryRowContext(ctx, query, symbol, rolesSlice).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to complete a QueryRowContext: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, err
}

func (pr *postgresMarketsRepo) DisableMarket(ctx context.Context, id uint64) (ok bool, err error) {

	query := `UPDATE markets SET enabled = false WHERE id = $1`

	tx, err := pr.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return false, fmt.Errorf("failed to ExecContext: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return false, errs.ErrMarketNotFound
	}

	err = tx.Commit()
	if err != nil {
		return false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return true, err
}

func (pr *postgresMarketsRepo) ViewMarketsByRoles(ctx context.Context, userRoles []string) ([]domain.Market, error) {

	query := `
		SELECT id, symbol, enabled, allowed_roles, deleted_at 
		FROM markets 
		WHERE deleted_at IS NULL 
		AND enabled = true
	`

	tx, err := pr.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to complete a QueryContext: %w", errs.ErrNoAvailableMarkets)
		}

		return nil, fmt.Errorf("failed to complete a QueryContext: %w", err)
	}
	defer rows.Close()

	var markets []domain.Market

	for rows.Next() {
		var m domain.Market

		var rolesSlice []string

		err := rows.Scan(&m.ID, &m.Symbol, &m.Enabled, pq.Array(&rolesSlice), &m.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market row: %w", err)
		}

		m.AllowedRoles = convertSliceToMap(rolesSlice)
		markets = append(markets, m)
	}

	Access := false
	result := make([]domain.Market, 0, len(markets))
	for _, m := range markets {
		if m.IsAccessibleForRoles(userRoles) {
			Access = true
			if m.IsAvailable() {
				result = append(result, m)
			}
		}
	}

	if !Access {
		return nil, errs.ErrMarketAccessDenied
	}

	if len(result) == 0 {
		return nil, errs.ErrNoAvailableMarkets
	}

	return result, nil
}

func convertToSlice(input string) []string {
	if input == "" {
		return []string{}
	}

	roles := strings.Split(input, ",")
	rolesSlice := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role != "" {
			rolesSlice = append(rolesSlice, role)
		}
	}
	return rolesSlice
}

func convertSliceToMap(rolesSlice []string) map[string]struct{} {
	rolesMap := make(map[string]struct{}, len(rolesSlice))
	for _, role := range rolesSlice {
		rolesMap[role] = struct{}{}
	}
	return rolesMap
}
