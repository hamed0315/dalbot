package services

import (
	"fmt"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/database"
)

// PreviousSettings saves the state before activating defense mode.
type PreviousSettings struct {
	Locks       map[string]bool   // lock_code -> was_active
	Punishments map[string]string // lock_code -> punishment
	FloodMaxMsgs int
}

var (
	defenseActive = make(map[int64]bool)
	defensePrev   = make(map[int64]PreviousSettings)
	defenseMu     sync.Mutex
)

// ActivateDefenseMode activates strict policies and announces the attack.
func ActivateDefenseMode(chatID int64, attackTypes []string) {
	defenseMu.Lock()
	if defenseActive[chatID] {
		defenseMu.Unlock()
		return
	}
	defenseActive[chatID] = true
	defenseMu.Unlock()

	// 1. Save current settings
	prev := saveCurrentSettings(chatID)
	defenseMu.Lock()
	defensePrev[chatID] = prev
	defenseMu.Unlock()

	// 2. Apply strict settings
	for _, lockType := range attackTypes {
		if lockType == "all" || lockType == "group" {
			continue
		}
		database.DB.Exec("INSERT OR REPLACE INTO locks (chat_id, lock_type, is_locked) VALUES (?, ?, 1)", chatID, lockType)
		SetLockPunishment(chatID, lockType, "ban", 1)
		InvalidateLocksCache(chatID)
	}
	// Increase flood sensitivity
	database.DB.Exec("UPDATE antispam_settings SET flood_max_msgs = 3 WHERE chat_id = ?", chatID)
	InvalidateConfig(chatID)

	// 3. Announce in group
	chatIDStr := fmt.Sprintf("%d", chatID)
	attackDesc := ""
	for _, t := range attackTypes {
		attackDesc += t + " + "
	}
	attackDesc = attackDesc[:len(attackDesc)-3] // remove trailing " + "
	msg := fmt.Sprintf(
		"🚨 **حالت دفاعی هجومی فعال شد!**\n\n"+
			"🛡️ DalBot حملهٔ ترکیبی (%s) را شناسایی کرد.\n"+
			"⚙️ قفل‌های لازم فعال و جریمه‌ها به «بن» تغییر یافت.\n"+
			"⏳ تا اطلاع ثانوی تنظیمات سخت‌گیرانه اعمال می‌شود.",
		attackDesc,
	)
	api.SendMessage(chatIDStr, msg)

	// 4. Log to super_owner
	LogToOwner("CRITICAL", fmt.Sprintf("🚨 حمله در گروه %d شناسایی شد: %s", chatID, attackDesc))
}

// DeactivateDefenseMode restores previous settings and announces peace.
func DeactivateDefenseMode(chatID int64) {
	defenseMu.Lock()
	if !defenseActive[chatID] {
		defenseMu.Unlock()
		return
	}
	defenseActive[chatID] = false
	prev, ok := defensePrev[chatID]
	if !ok {
		defenseMu.Unlock()
		return
	}
	delete(defensePrev, chatID)
	defenseMu.Unlock()

	// 1. Restore previous locks
	for lockCode, wasActive := range prev.Locks {
		if wasActive {
			database.DB.Exec("INSERT OR REPLACE INTO locks (chat_id, lock_type, is_locked) VALUES (?, ?, 1)", chatID, lockCode)
		} else {
			database.DB.Exec("DELETE FROM locks WHERE chat_id = ? AND lock_type = ?", chatID, lockCode)
		}
		if pun, ok := prev.Punishments[lockCode]; ok {
			SetLockPunishment(chatID, lockCode, pun, 1)
		}
	}
	InvalidateLocksCache(chatID)

	// Restore flood settings
	database.DB.Exec("UPDATE antispam_settings SET flood_max_msgs = ? WHERE chat_id = ?", prev.FloodMaxMsgs, chatID)
	InvalidateConfig(chatID)

	// 2. Announce peace
	chatIDStr := fmt.Sprintf("%d", chatID)
	msg := "🟢 **حالت نرمال تشخیص داده شد.**\n\n" +
		"✅ DalBot وضعیت گروه را ایمن ارزیابی کرد.\n" +
		"🔄 تمام تنظیمات به حالت قبل از حمله بازگردانده شد.\n" +
		"📊 گزارش کامل این حمله برای مالک گروه ارسال شد."
	api.SendMessage(chatIDStr, msg)

	// 3. Send report to group owner (and super_owner)
	SendWarReport(chatID)
}

// saveCurrentSettings captures current lock and antispam settings.
func saveCurrentSettings(chatID int64) PreviousSettings {
	prev := PreviousSettings{
		Locks:       make(map[string]bool),
		Punishments: make(map[string]string),
	}
	// Read locks
	rows, err := database.DB.Query("SELECT lock_type, is_locked FROM locks WHERE chat_id = ?", chatID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var lt string
			var active bool
			if err := rows.Scan(&lt, &active); err == nil {
				prev.Locks[lt] = active
				lp := GetLockPunishment(chatID, lt)
				prev.Punishments[lt] = lp.Punishment
			}
		}
	}
	// Read flood settings
	cfg := GetAntispamConfig(chatID)
	prev.FloodMaxMsgs = cfg.FloodMaxMsgs
	return prev
}

// SendWarReport generates and sends a war report to the group owner and super_owner.
func SendWarReport(chatID int64) {
	// For simplicity, we send to super_owner and also try to send to group owner
	ownerID := getGroupOwnerID(chatID)
	report := fmt.Sprintf(
		"📊 **گزارش حملهٔ گروه %d**\n"+
			"⏱ زمان: %s\n"+
			"🛡️ DalBot حمله را دفع کرد و گروه به حالت عادی بازگشت.\n"+
			"✅ وضعیت: امن",
		chatID, time.Now().Format("15:04:05"),
	)
	// Send to group owner
	if ownerID != 0 {
		api.SendMessage(fmt.Sprintf("%d", ownerID), report)
	}
	// Also to super_owner
	LogToOwner("INFO", report)
}

func getGroupOwnerID(chatID int64) int64 {
	var ownerID int64
	database.DB.QueryRow("SELECT owner_id FROM groups WHERE chat_id = ?", chatID).Scan(&ownerID)
	return ownerID
}
