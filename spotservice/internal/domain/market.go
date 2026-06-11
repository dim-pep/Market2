package domain

import "time"

type Market struct {
	ID           uint64
	Symbol       string
	Enabled      bool
	AllowedRoles map[string]struct{}
	DeletedAt    *time.Time
}

func (m *Market) IsAvailable() bool {
	return m.Enabled && (m.DeletedAt == nil || m.DeletedAt.IsZero())
}

func (m *Market) IsAccessibleForRoles(userRoles []string) bool {
	if len(m.AllowedRoles) == 0 {
		return true
	}

	for _, userRole := range userRoles {
		_, ok := m.AllowedRoles[userRole]
		if ok {
			return true
		}
	}

	return false
}




type ViewMarketsRequest struct {
	UserRoles []string
}

type ViewMarketsResponse struct {
	Markets []Market
}
