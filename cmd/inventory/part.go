package main

import "time"

type Part struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Quantity    int32     `json:"quantity"`
	Price       int64     `json:"price"`
	Reserved    int32     `json:"reserved"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreatePartRequest struct {
	Name        string  `json:"name"`
	Quantity    int32   `json:"quantity"`
	Price       int64   `json:"price"`
	Description *string `json:"description"`
}

type ListPartsParams struct {
	Name     string
	MinPrice *int64
	Limit    uint64
	Offset   uint64
}
