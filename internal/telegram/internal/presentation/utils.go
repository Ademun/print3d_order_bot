package presentation

import (
	"print3d-order-bot/internal/order"
)

func getStatusStr(status order.Status) string {
	switch status {
	case order.StatusActive:
		return "🟡 Активен"
	case order.StatusClosed:
		return "🟢 Закрыт"
	default:
		return "🔴 Неизвестен"
	}
}
