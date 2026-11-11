package main

// Russian interface texts.
// Not reviewed by a native speaker.
var messagesRU = map[string]string{
	// Page
	"dryRunBanner":  "dry-run: ничего не отправляется",
	"sessionOn":     "Сессия идёт",
	"sessionOff":    "Сессии нет",
	"weather":       "Погода",
	"tier":          "Уровень",
	"hull":          "Класс корабля",
	"fit":           "Фит",
	"allCans":       "Я собираю все контейнеры",
	"allCansHelp":   "Abyss Tracker записывает отправленные прохождения как «all cans», если отмечено, иначе как «Bioadaptive and some cans».",
	"fitNeedsToken": "укажите токен Abyss Tracker, чтобы увидеть свои фиты",
	"none":          "Нет",
	"startSession":  "Начать сессию",
	"updateSession": "Обновить сессию",
	"endSession":    "Завершить сессию",
	"settings":      "Настройки",
	"tokenLabel":    "Токен Abyss Tracker",
	"tokenSaved":    "сохранён; оставьте пустым, чтобы не менять",
	"tokenHowTo":    "создайте на abysstracker.com → My account → Personal access tokens",
	"logDirLabel":   "Папка журналов чата EVE",
	"logDirHelp":    "туда EVE записывает файлы Local_….txt",
	"language":      "Язык",
	"save":          "Сохранить",

	// Session
	"settingsSaved":  "Настройки сохранены.",
	"sessionStarted": "Сессия идёт: %s T%d. Скопируйте грузовой отсек перед первым прохождением.",
	"sessionEnded":   "Сессия завершена.",
	"errNoLogDir":    "Папка журналов чата EVE не найдена: %s (Настройки)",
	"errNoToken":     "Сначала укажите токен Abyss Tracker (Настройки)",

	// Local log
	"logFollowed":   "Отслеживается локальный журнал: %s.",
	"noLocalLog":    "В папке %s нет локального журнала: запустите EVE.",
	"logUnreadable": "Не удалось прочитать локальный журнал: %v",

	// Runs
	"baseSaved":           "Начальный груз сохранён.",
	"noBaseAtStart":       "Прохождение началось без копии грузового отсека: оно не будет отправлено.",
	"midRunIgnored":       "При запуске сессии прохождение уже шло: пропущено. Скопируйте грузовой отсек для следующего.",
	"runDoneCopy":         "Прохождение завершено. Скопируйте грузовой отсек (Ctrl+A, Ctrl+C).",
	"runLost":             "Прохождение проиграно (гибель в Бездне).",
	"movedRecopy":         "Смена системы: снова скопируйте грузовой отсек перед следующим прохождением.",
	"notSentNoBase":       "Прохождение не отправлено: нет копии грузового отсека в начале. Скопируйте его сейчас для следующего.",
	"notSentNewRun":       "Прохождение не отправлено: новое началось без копии грузового отсека.",
	"notSentMoved":        "Прохождение не отправлено: смена системы до копии грузового отсека.",
	"notSentStopped":      "Прохождение не отправлено: сессия завершена до копии грузового отсека.",
	"notSentUnknownItems": "Прохождение не отправлено: EVE не знает эти предметы (%s). Возможно, скопирован не грузовой отсек.",
	"notSentError":        "Прохождение не отправлено: %v",

	// Upload
	"dryRunStatus": "[dry-run] прохождение за %v не отправлено:\n%s",
	"dryRunNotify": "[dry-run] обнаружено прохождение за %v.",
	"sentStatus":   "Прохождение отправлено: %s",
	"sentNotify":   "Прохождение отправлено в Abyss Tracker.",
	"rejected":     "Abyss Tracker отклонил прохождение: %s",
	"sendFailed":   "Не удалось отправить, прохождение остаётся в очереди: %v",
}
