// auth.go
package main

// #############################################################################
// #                             YETKİLENDİRME MANTIĞI                           #
// #############################################################################
// Bu dosya, bota gelen isteklerin kim tarafından yapıldığını kontrol eden ve
// bu kullanıcının belirli komutları çalıştırma yetkisi olup olmadığını
// belirleyen temel güvenlik fonksiyonlarını içerir.
func isUserAdmin(userID int64) bool {
	// Gelen kullanıcı ID'sini, global yapılandırmadaki yönetici ID'si ile karşılaştırır.
	return userID == config.AdminChatID
}

// isUserAllowed, bir kullanıcının botu kullanma izni olup olmadığını kontrol eder.
// Bu fonksiyon, botun en temel güvenlik filtresidir. Bir kullanıcı, ya yönetici
// olmalı ya da .env dosyasındaki `ALLOWED_IDS` listesinde bulunmalıdır.
func isUserAllowed(userID int64) bool {
	// * Öncelik 1: Kullanıcı yönetici mi?
	// Eğer kullanıcı yönetici ise, başka bir kontrol yapmaya gerek kalmaz.
	if isUserAdmin(userID) {
		return true
	}

	// * Öncelik 2: Kullanıcı izin verilenler listesinde mi değil mi?
	// Yönetici değilse, global yapılandırmadaki `AllowedIDs` dilimini (slice) döngüye al.
	for _, allowedID := range config.AllowedIDs {
		// Eğer kullanıcının ID'si listedeki bir ID ile eşleşirse, izin verilir.
		if userID == allowedID {
			return true
		}
	}


	return false
}