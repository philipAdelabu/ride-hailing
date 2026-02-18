package i18n

// translations maps notification key → language code → format string.
// Format verbs follow fmt.Sprintf conventions.
//
// Supported languages: en (English), ru (Russian), tr (Turkish), tk (Turkmen).
var translations = map[string]map[string]string{

	// ─── Ride Requested (driver-facing) ──────────────────────────────────────
	"notification.ride.requested.title": {
		"en": "New Ride Request",
		"ru": "Новый запрос поездки",
		"tr": "Yeni Sürüş Talebi",
		"tk": "Täze Ýol Soragy",
	},
	// %s = pickup address
	"notification.ride.requested.body": {
		"en": "New ride request from %s",
		"ru": "Новый запрос поездки с адреса: %s",
		"tr": "%s adresinden yeni sürüş talebi",
		"tk": "%s salgydyndan täze ýol soragy",
	},
	"notification.ride.requested.sms": {
		"en": "New ride request nearby. Check your app!",
		"ru": "Новый запрос поездки рядом. Проверьте приложение!",
		"tr": "Yakında yeni sürüş talebi. Uygulamanıza bakın!",
		"tk": "Golaýda täze ýol soragy. Programmaňyzy barlaň!",
	},

	// ─── Ride Accepted (rider-facing) ────────────────────────────────────────
	"notification.ride.accepted.title": {
		"en": "Driver Found!",
		"ru": "Водитель найден!",
		"tr": "Sürücü Bulundu!",
		"tk": "Sürüji Tapyldy!",
	},
	// %s = driver name, %d = ETA minutes
	"notification.ride.accepted.body": {
		"en": "%s will pick you up in %d minutes",
		"ru": "%s заберёт вас через %d мин.",
		"tr": "%s sizi %d dakika içinde alacak",
		"tk": "%s sizi %d minutdan soň alar",
	},

	// ─── Ride Started (rider-facing) ─────────────────────────────────────────
	"notification.ride.started.title": {
		"en": "Ride Started",
		"ru": "Поездка начата",
		"tr": "Yolculuk Başladı",
		"tk": "Ýol Başlandy",
	},
	"notification.ride.started.body": {
		"en": "Your ride has started. Enjoy your trip!",
		"ru": "Ваша поездка началась. Приятного пути!",
		"tr": "Yolculuğunuz başladı. İyi yolculuklar!",
		"tk": "Ýoluňyz başlandy. Hoş ýol!",
	},

	// ─── Ride Completed — Rider ──────────────────────────────────────────────
	"notification.ride.completed.title.rider": {
		"en": "Ride Completed",
		"ru": "Поездка завершена",
		"tr": "Yolculuk Tamamlandı",
		"tk": "Ýol Tamamlandy",
	},
	// %s = formatted amount (e.g. "$12.50" or "12.50 TMT")
	"notification.ride.completed.body.rider": {
		"en": "Your ride is complete. Total fare: %s",
		"ru": "Ваша поездка завершена. Итоговая стоимость: %s",
		"tr": "Yolculuğunuz tamamlandı. Toplam ücret: %s",
		"tk": "Ýoluňyz tamamlandy. Jemi tölegi: %s",
	},

	// ─── Ride Completed — Driver ─────────────────────────────────────────────
	"notification.ride.completed.title.driver": {
		"en": "Ride Completed",
		"ru": "Поездка завершена",
		"tr": "Yolculuk Tamamlandı",
		"tk": "Ýol Tamamlandy",
	},
	// %s = formatted earnings amount
	"notification.ride.completed.body.driver": {
		"en": "Ride completed. You earned %s",
		"ru": "Поездка завершена. Вы заработали %s",
		"tr": "Yolculuk tamamlandı. %s kazandınız",
		"tk": "Ýol tamamlandy. Siz %s gazandyňyz",
	},

	// ─── Ride Cancelled ──────────────────────────────────────────────────────
	"notification.ride.cancelled.title": {
		"en": "Ride Cancelled",
		"ru": "Поездка отменена",
		"tr": "Yolculuk İptal Edildi",
		"tk": "Ýol Ýatyryldy",
	},
	// %s = who cancelled ("rider" / "driver" — should be pre-translated by caller)
	"notification.ride.cancelled.body": {
		"en": "Ride was cancelled by %s",
		"ru": "Поездка отменена пользователем: %s",
		"tr": "Yolculuk %s tarafından iptal edildi",
		"tk": "Ýol %s tarapyndan ýatyryldy",
	},
	"notification.ride.cancelled.by.rider": {
		"en": "rider",
		"ru": "пассажиром",
		"tr": "yolcu",
		"tk": "ýolagçy",
	},
	"notification.ride.cancelled.by.driver": {
		"en": "driver",
		"ru": "водителем",
		"tr": "sürücü",
		"tk": "sürüji",
	},

	// ─── Payment Received ────────────────────────────────────────────────────
	"notification.payment.received.title": {
		"en": "Payment Received",
		"ru": "Платёж получен",
		"tr": "Ödeme Alındı",
		"tk": "Töleg Alyndy",
	},
	// %s = formatted amount
	"notification.payment.received.body": {
		"en": "Payment of %s has been received",
		"ru": "Платёж на сумму %s получен",
		"tr": "%s tutarında ödeme alındı",
		"tk": "%s töleg alyndy",
	},

	// ─── Scheduled Ride Reminder ─────────────────────────────────────────────
	"notification.scheduled_ride.title": {
		"en": "Upcoming Ride Scheduled",
		"ru": "Предстоящая поездка",
		"tr": "Yaklaşan Planlanmış Yolculuk",
		"tk": "Meýilleşdirilen Ýol Ýakynlaşýar",
	},
	"notification.scheduled_ride.body": {
		"en": "Your scheduled ride is coming up soon. Get ready!",
		"ru": "Ваша запланированная поездка скоро начнётся. Приготовьтесь!",
		"tr": "Planlanmış yolculuğunuz yakında başlıyor. Hazır olun!",
		"tk": "Meýilleşdirilen ýoluňyz ýakynlaşýar. Taýyn boluň!",
	},

	// ─── Payout / Withdrawal ─────────────────────────────────────────────────
	"notification.payout.requested.title": {
		"en": "Withdrawal Requested",
		"ru": "Запрос на вывод средств",
		"tr": "Para Çekme Talebi",
		"tk": "Pul Çykarmak Soragy",
	},
	// %s = formatted amount
	"notification.payout.requested.body": {
		"en": "Your withdrawal of %s is being processed",
		"ru": "Ваш вывод средств на сумму %s обрабатывается",
		"tr": "%s tutarındaki para çekme işleminiz işleniyor",
		"tk": "%s mukdaryndaky pul çykarmak işleniýär",
	},
}
