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
	partColumnDesc      = "description"
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
		Columns(partColumnName, partColumnQuantity, partColumnPrice, partColumnReserved, partColumnDesc).
		Values(req.Name, req.Quantity, req.Price, 0, req.Description).
		Suffix("RETURNING " + partColumnID + ", " + partColumnCreatedAt).
		ToSql()
	if err != nil {
		return Part{}, err
	}

	part := Part{
		Name:        req.Name,
		Quantity:    req.Quantity,
		Price:       req.Price,
		Reserved:    0,
		Description: req.Description,
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
			partColumnDesc,
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
		&part.Description,
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

func (r *partRepository) List(ctx context.Context, params ListPartsParams) ([]Part, error) {
	builder := psql.
		Select(
			partColumnID,
			partColumnName,
			partColumnQuantity,
			partColumnPrice,
			partColumnReserved,
			partColumnDesc,
			partColumnCreatedAt,
		).
		From(partsTableName).
		OrderBy(partColumnID + " ASC").
		Limit(params.Limit).
		Offset(params.Offset)

	if params.Name != "" {
		builder = builder.Where(sq.Eq{partColumnName: params.Name})
	}

	if params.MinPrice != nil {
		builder = builder.Where(sq.GtOrEq{partColumnPrice: *params.MinPrice})
	}

	query, args, err := builder.ToSql()
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
			&part.Description,
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
