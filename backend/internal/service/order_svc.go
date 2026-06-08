package service

import (
	"context"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
)

type OrderSvc struct {
	repo *repository.OrderRepo
}

func NewOrderSvc(repo *repository.OrderRepo) *OrderSvc {
	return &OrderSvc{repo: repo}
}

func (s *OrderSvc) CreateOrder(ctx context.Context, order *model.Order) error {
	return s.repo.CreateOrder(ctx, order)
}

// GetByID returns the order with remaining payment seconds calculated.
func (s *OrderSvc) GetByID(ctx context.Context, id uint64) (*model.Order, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.fillRemaining(order)
	return order, nil
}

func (s *OrderSvc) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	order, err := s.repo.GetByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	s.fillRemaining(order)
	return order, nil
}

func (s *OrderSvc) ListByBuyer(ctx context.Context, buyerID uint64, page model.PageRequest) ([]model.Order, int64, error) {
	orders, total, err := s.repo.ListByBuyer(ctx, buyerID, page)
	if err != nil {
		return nil, 0, err
	}
	for i := range orders {
		s.fillRemaining(&orders[i])
	}
	return orders, total, nil
}

func (s *OrderSvc) ListByBuyerWithDetails(ctx context.Context, buyerID uint64, page model.PageRequest) ([]model.SellerOrderItem, int64, error) {
	items, total, err := s.repo.ListByBuyerWithDetails(ctx, buyerID, page)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		s.fillRemaining(&items[i].Order)
	}
	return items, total, nil
}

func (s *OrderSvc) ListBySeller(ctx context.Context, sellerID uint64, page model.PageRequest) ([]model.Order, int64, error) {
	orders, total, err := s.repo.ListBySeller(ctx, sellerID, page)
	if err != nil {
		return nil, 0, err
	}
	for i := range orders {
		s.fillRemaining(&orders[i])
	}
	return orders, total, nil
}

func (s *OrderSvc) ListBySellerWithDetails(ctx context.Context, sellerID uint64, page model.PageRequest, statusFilter ...int) ([]model.SellerOrderItem, int64, error) {
	items, total, err := s.repo.ListBySellerWithDetails(ctx, sellerID, page, statusFilter...)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		s.fillRemaining(&items[i].Order)
	}
	return items, total, nil
}

func (s *OrderSvc) ConfirmAddress(ctx context.Context, orderID uint64, address string) error {
	return s.repo.ConfirmAddress(ctx, orderID, address)
}

// Pay simulates a payment: marks the order as paid and creates a payment record.
func (s *OrderSvc) Pay(ctx context.Context, orderID uint64) error {
	return s.repo.MarkAsPaid(ctx, orderID)
}

func (s *OrderSvc) RevenueBySeller(ctx context.Context, sellerID uint64) (string, error) {
	return s.repo.RevenueBySeller(ctx, sellerID)
}

func (s *OrderSvc) CountByStatus(ctx context.Context, sellerID uint64) (map[int]int64, error) {
	return s.repo.CountByStatus(ctx, sellerID)
}

func (s *OrderSvc) fillRemaining(order *model.Order) {
	if order.Status != model.OrderStatusUnpaid || order.ExpiresAt == nil {
		return
	}
	sec := order.ExpiresAt.Unix() - time.Now().Unix()
	if sec < 0 {
		sec = 0
	}
	order.RemainingSec = sec
}

func (s *OrderSvc) Ship(ctx context.Context, orderID uint64) error {
	return s.repo.UpdateStatus(ctx, orderID, model.OrderStatusPaid, model.OrderStatusShipped)
}

func (s *OrderSvc) ConfirmReceipt(ctx context.Context, orderID uint64) error {
	return s.repo.UpdateStatus(ctx, orderID, model.OrderStatusShipped, model.OrderStatusCompleted)
}
