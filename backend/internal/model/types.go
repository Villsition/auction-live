package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ============================================================
// Custom types for JSON columns
// ============================================================

type StringArray []string

func (s *StringArray) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("StringArray.Scan: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, s)
}

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

type JSONMap map[string]any

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("JSONMap.Scan: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, m)
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// ============================================================
// User role / status
// ============================================================

type UserRole uint8

const (
	UserRoleNormal UserRole = 0
	UserRoleSeller UserRole = 1
	UserRoleAdmin  UserRole = 2
)

type UserStatus uint8

const (
	UserStatusDisabled UserStatus = 0
	UserStatusNormal   UserStatus = 1
)

// ============================================================
// Category status
// ============================================================

type CategoryStatus uint8

const (
	CategoryStatusDisabled CategoryStatus = 0
	CategoryStatusNormal   CategoryStatus = 1
)

// ============================================================
// Product status
// ============================================================

type ProductStatus uint8

const (
	ProductStatusDraft     ProductStatus = 0
	ProductStatusListed    ProductStatus = 1
	ProductStatusBidding   ProductStatus = 2
	ProductStatusSold      ProductStatus = 3
	ProductStatusUnsold    ProductStatus = 4
	ProductStatusCancelled ProductStatus = 5
)

func (s ProductStatus) String() string {
	switch s {
	case ProductStatusDraft:
		return "draft"
	case ProductStatusListed:
		return "listed"
	case ProductStatusBidding:
		return "bidding"
	case ProductStatusSold:
		return "sold"
	case ProductStatusUnsold:
		return "unsold"
	case ProductStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// ============================================================
// Live room status
// ============================================================

type LiveRoomStatus uint8

const (
	LiveRoomStatusOffline LiveRoomStatus = 0
	LiveRoomStatusLive    LiveRoomStatus = 1
	LiveRoomStatusEnded   LiveRoomStatus = 2
)

// ============================================================
// Auction session status
// ============================================================

type AuctionStatus uint8

const (
	AuctionStatusPending   AuctionStatus = 0
	AuctionStatusActive    AuctionStatus = 1
	AuctionStatusSold      AuctionStatus = 2
	AuctionStatusUnsold    AuctionStatus = 3
	AuctionStatusCancelled AuctionStatus = 4
)

// ============================================================
// Order status
// ============================================================

type OrderStatus uint8

const (
	OrderStatusUnpaid    OrderStatus = 0
	OrderStatusPaid      OrderStatus = 1
	OrderStatusShipped   OrderStatus = 2
	OrderStatusCompleted OrderStatus = 3
	OrderStatusCancelled OrderStatus = 4
	OrderStatusRefunded  OrderStatus = 5
)

// ============================================================
// Payment status
// ============================================================

type PaymentStatus uint8

const (
	PaymentStatusPending  PaymentStatus = 0
	PaymentStatusSuccess  PaymentStatus = 1
	PaymentStatusFailed   PaymentStatus = 2
	PaymentStatusRefunded PaymentStatus = 3
)

// ============================================================
// Notification type
// ============================================================

type NotificationType uint8

const (
	NotifTypeSystem  NotificationType = 0
	NotifTypeRemind  NotificationType = 1
	NotifTypeOutbid  NotificationType = 2
	NotifTypeDeal    NotificationType = 3
)
