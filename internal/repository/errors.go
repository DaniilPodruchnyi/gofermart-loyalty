package repository

import "errors"

var (
	ErrLoginExists       = errors.New("login already exists")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrBalanceNotFound   = errors.New("balance not found")
)
