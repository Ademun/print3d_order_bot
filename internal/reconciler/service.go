package reconciler

import (
	"context"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/order"
)

type Service interface {
	Start(ctx context.Context) error
}

type DefaultService struct {
	orderService order.Service
	fileService  file.Service
}

func NewDefaultService(orderService order.Service, fileService file.Service) Service {
	return &DefaultService{
		orderService: orderService,
		fileService:  fileService,
	}
}

func (d *DefaultService) Start(ctx context.Context) error {

}

func (d *DefaultService) ReconcileOrder(ctx context.Context, order order.DBOrder) {
	orderFiles := d.orderService
}
