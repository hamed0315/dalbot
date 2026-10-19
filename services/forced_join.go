package services

import (
	"fmt"
	"sync"
	"time"

	"dalbot/api"
	"dalbot/config"
	"dalbot/database"
)

type ForcedChannel struct {
	Username string
	Title    string
	Link     string
}

type cacheEntry struct {
	isMember bool
	expires  time.Time
}

var membershipCache = struct {
	sync.Mutex
	data map[string]cacheEntry
}{data: make(map[string]cacheEntry)}

func GetForcedChannels(chatID int64) ([]ForcedChannel, error) {
	rows, err := database.DB.Query(
		"SELECT channel_username, channel_title, join_link FROM group_forced_channels WHERE chat_id = ?", chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var channels []ForcedChannel
	for rows.Next() {
		var ch ForcedChannel
		if err := rows.Scan(&ch.Username, &ch.Title, &ch.Link); err != nil {
			continue
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func CheckAllMissingChannels(chatID int64, userID int) []ForcedChannel {
	channels, err := GetForcedChannels(chatID)
	if err != nil || len(channels) == 0 {
		return nil
	}

	var missing []ForcedChannel
	for _, ch := range channels {
		if !isMemberCached(chatID, userID, ch.Username) {
			missing = append(missing, ch)
		}
	}
	return missing
}

func isMemberCached(chatID int64, userID int, channelUsername string) bool {
	key := fmt.Sprintf("%d:%d:%s", chatID, userID, channelUsername)
	membershipCache.Lock()
	entry, exists := membershipCache.data[key]
	if exists && time.Now().Before(entry.expires) {
		membershipCache.Unlock()
		return entry.isMember
	}
	membershipCache.Unlock()

	isMember := IsUserInChannel(channelUsername, userID)

	membershipCache.Lock()
	membershipCache.data[key] = cacheEntry{isMember: isMember, expires: time.Now().Add(10 * time.Second)}
	membershipCache.Unlock()
	return isMember
}

func IsUserInChannel(channelUsername string, userID int) bool {
	cm, err := api.GetChatMember(channelUsername, userID)
	if err != nil {
		return false
	}
	return cm.Status == "member" || cm.Status == "administrator" || cm.Status == "creator"
}

func IsBotMemberOfChannel(channelUsername string) bool {
	return IsUserInChannel(channelUsername, config.BotID)
}

func GetJoinLinkForChannel(username string) (string, error) {
	chat, err := api.GetChat(username)
	if err != nil {
		return "", err
	}
	if chat.InviteLink != "" {
		return chat.InviteLink, nil
	}
	link, err := api.ExportChatInviteLink(username)
	if err != nil {
		return "", err
	}
	return link, nil
}
