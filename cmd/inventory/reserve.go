package main

import (
	"context"
	"errors"
)

var (
	errInsufficientStock     = errors.New("недостаточно остатков")
	errInvalidReserveRequest = errors.New("количество для резерва должно быть положительным")
)

func (a *inventoryApp) reservePart(ctx context.Context, id string, qty int32) error {
	if qty <= 0 {
		return errInvalidReserveRequest
	}

	part, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if part.AvailableQuantity() < qty {
		return errInsufficientStock
	}

	return a.repo.IncrementReserved(ctx, id, qty)
}
