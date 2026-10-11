package main

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	psql            = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	errPartNotFound = errors.New("деталь не найдена")
)

const (
	partsTableName      = "parts"
	partColumnID        = "id"
	partColumnName      = "name"
	partColumnQuantity  = "quantity"
	partColumnPrice     = "price"
	partColumnReserved  = "reserved"
	partColumnCreatedAt = "created_at"
)

type partRepository struct {
	db *pgxpool.Pool
}

func newPartRepository(db *pgxpool.Pool) *partRepository {
	return &partRepository{db: db}
}

func (r *partRepository) Create(ctx context.Context, req CreatePartRequest) (Part, error) {
	query, args, err := psql.
		Insert(partsTableName).
		Columns(partColumnName, partColumnQuantity, partColumnPrice, partColumnReserved).
		Values(req.Name, req.Quantity, req.Price, 0).
		Suffix("RETURNING " + partColumnID + ", " + partColumnCreatedAt).
		ToSql()
	if err != nil {
		return Part{}, err
	}

	part := Part{
		Name:     req.Name,
		Quantity: req.Quantity,
		Price:    req.Price,
		Reserved: 0,
	}

	err = r.db.QueryRow(ctx, query, args...).Scan(
		&part.ID,
		&part.CreatedAt,
	)
	if err != nil {
		return Part{}, err
	}

	return part, nil
}

func (r *partRepository) GetByID(ctx context.Context, id string) (Part, error) {
	query, args, err := psql.
		Select(
			partColumnID,
			partColumnName,
			partColumnQuantity,
			partColumnPrice,
			partColumnReserved,
			partColumnCreatedAt,
		).
		From(partsTableName).
		Where(sq.Eq{partColumnID: id}).
		ToSql()
	if err != nil {
		return Part{}, err
	}

	var part Part
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&part.ID,
		&part.Name,
		&part.Quantity,
		&part.Price,
		&part.Reserved,
		&part.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Part{}, errPartNotFound
		}

		return Part{}, err
	}

	return part, nil
}

func (r *partRepository) List(ctx context.Context) ([]Part, error) {
	query, args, err := psql.
		Select(
			partColumnID,
			partColumnName,
			partColumnQuantity,
			partColumnPrice,
			partColumnReserved,
			partColumnCreatedAt,
		).
		From(partsTableName).
		OrderBy(partColumnID + " ASC").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []Part
	for rows.Next() {
		var part Part
		if err = rows.Scan(
			&part.ID,
			&part.Name,
			&part.Quantity,
			&part.Price,
			&part.Reserved,
			&part.CreatedAt,
		); err != nil {
			return nil, err
		}

		parts = append(parts, part)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return parts, nil
}
