package services

import "dalbot/database"

func WarmUpCaches() {
    rows, err := database.DB.Query("SELECT chat_id FROM antispam_settings")
    if err != nil {
        return
    }
    defer rows.Close()
    for rows.Next() {
        var chatID int64
        if err := rows.Scan(&chatID); err == nil {
            GetAntispamConfig(chatID) // این فراخوانی کش را برای این گروه می‌سازد
        }
    }
}
