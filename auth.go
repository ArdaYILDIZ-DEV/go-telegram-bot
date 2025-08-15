// auth.go
package main

// #############################################################################
// #                             YETKİLENDİRME MANTIĞI                           #
// #############################################################################
// Bu dosya, bota gelen isteklerin kim tarafından yapıldığını kontrol eden ve
// bu kullanıcının belirli komutları çalıştırma yetkisi olup olmadığını
// belirleyen temel güvenlik fonksiyonlarını içerir.

// isUserAdmin, bir kullanıcının botun ana yöneticisi olup olmadığını doğrular.
// Yönetici, .env dosyasında belirtilen `ADMIN_CHAT_ID` ile eşleşen kişidir
// ve tüm kısıtlamalardan muaftır.
//
// Parametreler:
//   userID (int64): Kontrol edilecek kullanıcının Telegram ID'si.
//
// Dönen Değer:
//   (bool): Kullanıcı yönetici ise `true`, değilse `false`.
func isUserAdmin(userID int64) bool {
	// Gelen kullanıcı ID'sini, global yapılandırmadaki yönetici ID'si ile karşılaştırır.
	return userID == config.AdminChatID
}

// isUserAllowed, bir kullanıcının botu kullanma izni olup olmadığını kontrol eder.
// Bu fonksiyon, botun en temel güvenlik filtresidir. Bir kullanıcı, ya yönetici
// olmalı ya da .env dosyasındaki `ALLOWED_IDS` listesinde bulunmalıdır.
//
// Bu fonksiyon, herhangi bir komut işlenmeden önce çağrılır.
//
// Parametreler:
//   userID (int64): Kontrol edilecek kullanıcının Telegram ID'si.
//
// Dönen Değer:
//   (bool): Kullanıcının botu kullanma izni varsa `true`, yoksa `false`.
func isUserAllowed(userID int64) bool {
	// * Öncelik 1: Kullanıcı yönetici mi?
	// Eğer kullanıcı yönetici ise, başka bir kontrol yapmaya gerek kalmaz.
	if isUserAdmin(userID) {
		return true
	}

	// * Öncelik 2: Kullanıcı izin verilenler listesinde mi?
	// Yönetici değilse, global yapılandırmadaki `AllowedIDs` dilimini (slice) döngüye al.
	for _, allowedID := range config.AllowedIDs {
		// Eğer kullanıcının ID'si listedeki bir ID ile eşleşirse, izin verilir.
		if userID == allowedID {
			return true
		}
	}

	// * Sonuç: Kullanıcı hiçbir koşulu sağlamadı.
	// Döngü bitti ve eşleşme bulunamadıysa, kullanıcı yetkisizdir.
	return false
}