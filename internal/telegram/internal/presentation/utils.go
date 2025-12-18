package presentation

import (
	"fmt"
	"math"
	"print3d-order-bot/internal/order"
	"strconv"
	"strings"
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

func FormatRUB(amount float32) string {
	rounded := math.Round(float64(amount)*100) / 100

	intPart := int64(rounded)
	fracPart := int(math.Round((rounded - float64(intPart)) * 100))

	intStr := formatWithThousandsSeparator(intPart)

	if fracPart == 0 {
		return intStr
	}

	fracStr := strconv.Itoa(fracPart)
	if len(fracStr) == 1 {
		fracStr = "0" + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")

	return intStr + "," + fracStr
}

func formatWithThousandsSeparator(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}

	str := strconv.FormatInt(n, 10)

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(' ')
		}
		result.WriteRune(digit)
	}

	if negative {
		return "-" + result.String()
	}

	return result.String()
}

func ParseRUB(input string) (float32, error) {
	s := strings.TrimSpace(input)
	s = strings.ReplaceAll(s, "₽", "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	val, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse currency string: %w", err)
	}
	return float32(val), nil
}
