package main

import "time"

type Part struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Quantity  int32     `json:"quantity"`
	Price     int64     `json:"price"`
	Reserved  int32     `json:"reserved"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePartRequest struct {
	Name     string `json:"name"`
	Quantity int32  `json:"quantity"`
	Price    int64  `json:"price"`
}
