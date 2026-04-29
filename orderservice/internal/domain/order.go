package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderType int

const (
	OrderTypeUnspecified OrderType = iota
	OrderTypeMarketOrder
	OrderTypeLimitOrder
)

type OrderStatus int

const (
	OrderStatusUnspecified OrderStatus = iota
	OrderStatusCreated
	OrderStatusActive
	OrderStatusComplete
	OrderStatusFailed
)

type Order struct {
	orderID     string
	orderStatus OrderStatus
	userID      string
	marketID    string
	sessionID   string
	orderType   OrderType
	price       decimal.Decimal
	quantity    decimal.Decimal
	createdAt   time.Time
}

func NewOrder(orderID, userID, sessionID, marketID string, orderType OrderType, orderStatus OrderStatus, price float64, quantity float64, createdAt time.Time) *Order {

	order := &Order{
		orderID:     orderID,
		userID:      userID,
		sessionID:   sessionID,
		marketID:    marketID,
		orderType:   orderType,
		orderStatus: orderStatus,
		createdAt:   createdAt,
		price:       decimal.NewFromFloat(price),
		quantity:    decimal.NewFromFloat(quantity),
	}

	return order
}
