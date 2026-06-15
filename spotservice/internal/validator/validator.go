package validator

import (
	"fmt"

	"github.com/dim-pep/Market2/proto/pb/spot_service"
)

const viewMarketsValidationCapacity = 1

// Validator accumulates request validation errors.
//
// Usage:
//
//	validationErrors, ok := validator.NewValidator().
//		ValidateViewMarkets(req).
//		Validate()
type Validator struct {
	errors []string
}

// NewValidator creates an empty validator.
func NewValidator() *Validator {
	return &Validator{
		errors: make([]string, 0, viewMarketsValidationCapacity),
	}
}

func (v *Validator) ValidateViewMarkets(req *spot_service.ViewMarketsRequest) *Validator {
	if req == nil {
		v.errors = append(v.errors, "request is required")
		return v
	}

	for _, role := range req.UserRoles {
		if role == "" {
			continue
		}

		if _, ok := RolesMap[role]; !ok {
			v.errors = append(v.errors, fmt.Sprintf("unknown role: %s", role))
		}
	}

	return v
}

// Validate returns accumulated validation errors and whether validation passed.
func (v *Validator) Validate() ([]string, bool) {
	if v == nil {
		return []string{"validator is not initialized"}, false
	}

	return v.errors, len(v.errors) == 0
}

var RolesMap = map[string]struct{}{
	"admin":  {},
	"trader": {},
	"viewer": {},
}
