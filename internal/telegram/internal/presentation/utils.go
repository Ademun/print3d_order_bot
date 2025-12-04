package presentation

import "print3d-order-bot/internal/pkg/model"

func getStatusStr(status model.OrderStatus) string {
	switch status {
	case model.StatusActive:
		return "🟡 Активен"
	case model.StatusClosed:
		return "🟢 Закрыт"
	default:
		return "🔴 Неизвестен"
	}
}
