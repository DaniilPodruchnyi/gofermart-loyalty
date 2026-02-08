package models

import "time"

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

type Order struct {
	ID         int64       `db:"id"`
	UserID     int64       `db:"user_id"`
	Number     string      `db:"number"`
	Status     OrderStatus `db:"status"`
	Accrual    float64     `db:"accrual"`
	UploadedAt time.Time   `db:"uploaded_at"`
}

func (o *Order) GetID() int64 {
	return o.ID
}

func (o *Order) SetID(id int64) {
	o.ID = id
}

type OrderResponse struct {
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"`
	UploadedAt string      `json:"uploaded_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
