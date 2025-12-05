package presentation

import (
	"fmt"
	"print3d-order-bot/internal/pkg/model"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"strconv"
	"strings"
)

func GenericErrorMsg() string {
	return "*❌ Произошла неизвестная ошибка, попробуйте позже*"
}

func HelpMsg() string {
	var sb strings.Builder
	sb.WriteString("*❓ Чтобы создать заказ отправь или перешли боту сообщение с вложениями и/или ссылкой / почтой / номером телефона*")
	sb.WriteString(breakLine(2))
	sb.WriteString("*🤖 Бот поддерживает следующие вложения: фото, видео, файлы, кружочки и голосовые сообщения*")
	sb.WriteString(breakLine(2))
	sb.WriteString("*⚙️ Доступные команды:*")
	sb.WriteString(breakLine(2))
	sb.WriteString("*/orders — просмотреть активные заказы*")
	return sb.String()
}

func AskOrderTypeMsg() string {
	return "*❓ Вы хотите создать новый заказ или добавить информацию к старому?*"
}

func AddedDataToOrderMsg() string {
	return "*✔️ Добавлены новые данные к заказу*"
}

func AskClientNameMsg() string {
	return "*👤 Введите имя клиента*"
}

func AskOrderCostMsg() string {
	return "*💰 Введите стоимость заказа в рублях*"
}

func CostValidationErrorMsg() string {
	return "❌ Стоимость заказа должна быть числом"
}

func AskOrderCommentsMsg() string {
	return "*💬 Введите комментарий к заказу*"
}

func AskOrderSelectionMsg() string {
	return "*📝 Выберите заказ из списка*"
}

func NewOrderPreviewMsg(data *fsm.OrderData) string {
	var sb strings.Builder
	sb.WriteString("*❓ Создать новый заказ?*")
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("*👤 Клиент: %s*", escapeMarkdown(data.ClientName)))
	sb.WriteString(breakLine(2))
	costStr := strconv.FormatFloat(float64(data.Cost), 'f', -1, 64)
	sb.WriteString(fmt.Sprintf("*💲 Стоимость заказа %s₽*", escapeMarkdown(costStr)))
	if len(data.Comments) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*💬 Комментарии к заказу:*")
		for _, comment := range data.Comments {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(comment)))
		}
	}
	if len(data.Contacts) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*📞 Контакты:*")
		for _, contact := range data.Contacts {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(contact)))
		}
	}
	if len(data.Links) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*🔗 Ссылки:*")
		for _, link := range data.Links {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(link)))
		}
	}
	if len(data.Files) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*📄 Файлы:*")
		for _, file := range data.Files {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(file.FileName)))
		}
	}
	return sb.String()
}

func NewOrderCancelledMsg() string {
	return "*❌ Создание заказа отменено*"
}

func NewOrderCreatedMsg() string {
	return "*✔️ Заказ успешно создан*"
}

func OrderViewMsg(data *model.Order) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*Заказ №%d от %s*", data.OrderID, escapeMarkdown(data.CreatedAt.Format("2006-01-02"))))
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("*Статус: %s*", getStatusStr(data.OrderStatus)))
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("*👤 Клиент: %s*", escapeMarkdown(data.ClientName)))
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("*💲 Стоимость заказа %f₽*", data.Cost))
	if len(data.Comments) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*💬 Комментарии к заказу:*")
		for _, comment := range data.Comments {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(comment)))
		}
	}
	if len(data.Contacts) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*📞 Контакты:*")
		for _, contact := range data.Contacts {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(contact)))
		}
	}
	if len(data.Links) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*🔗 Ссылки:*")
		for _, link := range data.Links {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(link)))
		}
	}
	if len(data.Filenames) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("*📄 Файлы:*")
		for _, name := range data.Filenames {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("*%s*", escapeMarkdown(name)))
		}
	}
	return sb.String()
}

func EmptyOrderListMsg() string {
	return "*🔍 У вас пока нет активных заказов*"
}

func breakLine(n int) string {
	return strings.Repeat("\n", n)
}

func escapeMarkdown(s string) string {
	specialChars := []string{
		"_", "*", "[", "]", "(", ")", "~", "`", ">",
		"#", "+", "-", "=", "|", "{", "}", ".", "!",
	}

	result := s
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}

	return result
}
