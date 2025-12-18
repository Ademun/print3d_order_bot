package presentation

import (
	"fmt"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"strings"
)

func GenericErrorMsg() string {
	return "<b>❌ Произошла неизвестная ошибка, попробуйте позже</b>"
}

func StateConversionErrorMsg() string {
	return "<b>❌ Не удалось загрузить данные прошлого ответа. Начните сначала</b>"
}

func OrderIDsLoadErrorMsg() string {
	return "<b>❌ Не удалось загрузить список заказов. Попробуйте позже<b/>"
}

func OrderLoadErrorMsg() string {
	return "<b>❌ Не удалось загрузить данные о заказе. Попробуйте позже</b>"
}

func AddFilesToOrderWarningMsg() string {
	return "<b>⚠️ Не удалось добавить файлы к заказу. Они будут добавлены при загрузке файлов заказа</b>"
}

func OrderCreationErrorMsg() string {
	return "<b>❌ Не удалось создать заказ. Попробуйте снова</b>"
}

func OrderCloseErrorMsg() string {
	return "<b>❌ Не удалось закрыть заказ. Попробуйте позже</b>"
}

func OrderRestoreErrorMsg() string {
	return "<b>❌ Не удалось восстановить заказ. Попробуйте позже</b>"
}

func FilesLoadErrorMsg() string {
	return "<b>❌ Не удалось загрузить файлы заказа. Попробуйте позже</b>"
}

func HelpMsg() string {
	var sb strings.Builder
	sb.WriteString("<b>❓ Чтобы создать заказ отправь или перешли боту сообщение с вложениями и/или ссылкой / почтой / номером телефона</b>")
	sb.WriteString(breakLine(2))
	sb.WriteString("<b>🤖 Бот поддерживает следующие вложения: фото, видео, файлы, кружочки и голосовые сообщения</b>")
	sb.WriteString(breakLine(2))
	sb.WriteString("<b>⚙️ Доступные команды:</b>")
	sb.WriteString(breakLine(2))
	sb.WriteString("<b>/orders — просмотреть активные заказы</b>")
	return sb.String()
}

func AskOrderTypeMsg() string {
	return "<b>❓ Вы хотите создать новый заказ или добавить информацию к старому?</b>"
}

func AddedDataToOrderMsg() string {
	return "<b>✔️ Добавлены новые данные к заказу</b>"
}

func AskClientNameMsg() string {
	return "<b>👤 Введите имя клиента</b>"
}

func AskOrderCostMsg() string {
	return "<b>💰 Введите стоимость заказа в рублях</b>"
}

func CostValidationErrorMsg() string {
	return "❌ Стоимость заказа должна быть числом"
}

func AskOrderCommentsMsg() string {
	return "<b>💬 Введите комментарий к заказу</b>"
}

func AskOrderSelectionMsg() string {
	return "<b>📝 Выберите заказ из списка</b>"
}

func StartingDownloadMsg(total int) string {
	return fmt.Sprintf("<b>💾 Начинаю загрузку файлов. Всего файлов: %d</b>", total)
}

func DownloadProgressMsg(fileName string, progress int, total int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>💾 Загружено %d файлов из %d</b>", progress, total))
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("Загружаю файл `%s...`", fileName))
	return sb.String()
}

func DownloadResultMsg(errors map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>✔️ Загрузка файлов завершена</b>"))
	if len(errors) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString(fmt.Sprintf("<b>❌ Не удалось загрузить %d файлов</b>", len(errors)))
		for filename, err := range errors {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("%s - %s", filename, err))
		}
	}
	return sb.String()
}

func NewOrderPreviewMsg(data *fsm.OrderData) string {
	var sb strings.Builder
	sb.WriteString("<b>❓ Создать новый заказ?</b>")
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("<b>👤 Клиент: %s</b>", data.ClientName))
	sb.WriteString(breakLine(2))
	costStr := FormatRUB(data.Cost)
	sb.WriteString(fmt.Sprintf("<b>💲 Стоимость заказа %s₽</b>", costStr))
	if len(data.Comments) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>💬 Комментарии к заказу:</b>")
		for _, comment := range data.Comments {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", comment))
		}
	}
	if len(data.Contacts) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>📞 Контакты:</b>")
		for _, contact := range data.Contacts {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", contact))
		}
	}
	if len(data.Links) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>🔗 Ссылки:</b>")
		for _, link := range data.Links {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", link))
		}
	}
	if len(data.Files) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>📄 Файлы:</b>")
		for _, file := range data.Files {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", file.Name))
		}
	}
	return sb.String()
}

func NewOrderCancelledMsg() string {
	return "<b>❌ Создание заказа отменено</b>"
}

func NewOrderCreatedMsg() string {
	return "<b>✔️ Заказ успешно создан</b>"
}

func OrderViewMsg(data *order.ResponseOrder) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Заказ №%d от %s</b>", data.ID, data.CreatedAt.Format("2006-01-02")))
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("<b>Статус: %s</b>", getStatusStr(data.Status)))
	sb.WriteString(breakLine(2))
	sb.WriteString(fmt.Sprintf("<b>👤 Клиент: %s</b>", data.ClientName))
	sb.WriteString(breakLine(2))
	costStr := FormatRUB(data.Cost)
	sb.WriteString(fmt.Sprintf("<b>💲 Стоимость заказа %s₽</b>", costStr))
	if len(data.Comments) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>💬 Комментарии к заказу:</b>")
		for _, comment := range data.Comments {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", comment))
		}
	}
	if len(data.Contacts) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>📞 Контакты:</b>")
		for _, contact := range data.Contacts {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", contact))
		}
	}
	if len(data.Links) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>🔗 Ссылки:</b>")
		for _, link := range data.Links {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", link))
		}
	}
	if len(data.Files) > 0 {
		sb.WriteString(breakLine(2))
		sb.WriteString("<b>📄 Файлы:</b>")
		for _, file := range data.Files {
			sb.WriteString(breakLine(1))
			sb.WriteString(fmt.Sprintf("<b>%s</b>", file.Name))
		}
	}
	return sb.String()
}

func EmptyOrderListMsg() string {
	return "<b>🔍 У вас пока нет активных заказов</b>"
}

func PendingDownloadMsg() string {
	return "<b>Пожалуйста, дождитесь загрузки файлов</b>"
}

func PendingUploadMsg() string {
	return "<b>Пожалуйста, дождитесь отправки файлов</b>"
}

func UploadErrorMsg(filename string) string {
	return fmt.Sprintf("<b>❌ Не удалось загрузить файл %s</b>", filename)
}

func breakLine(n int) string {
	return strings.Repeat("\n", n)
}
