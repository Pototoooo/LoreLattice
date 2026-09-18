package billing

import (
	"errors"

	apperrors "github.com/Pototoooo/lorelattice/internal/errors"
)

// ToAppError converts a provider-boundary billing error into LoreLattice's
// normal middleware envelope. The stable string code remains in details so
// both REST and SSE callers can make the same recovery decision.
func ToAppError(err error) error {
	var billingErr *BillingError
	if !errors.As(err, &billingErr) {
		return err
	}
	code := apperrors.ErrServiceUnavailable
	if billingErr.HTTPStatus == 402 {
		code = apperrors.ErrBillingPaymentRequired
	} else if billingErr.HTTPStatus == 409 {
		code = apperrors.ErrConflict
	}
	return &apperrors.AppError{
		Code: code, Message: billingErr.Message, HTTPCode: billingErr.HTTPStatus,
		Details: map[string]any{
			"code": billingErr.Code, "feature": billingErr.Feature, "reset_at": billingErr.ResetAt,
		},
	}
}
