package services

import (
	"strings"
	"time"

	"dalbot/api"
	"dalbot/config"
	"dalbot/logger"
)

// isUserInChannelWithRetry checks membership with up to 3 retries on network/server errors.
// If the API returns "no such group or user" OR "permission_denied", it immediately returns false (user not member).
// Returns true only if confirmed member/administrator/creator, or if network error persists (safe default).
func isUserInChannelWithRetry(channelUsername string, userID int) bool {
	// Ensure @ prefix for channel usernames
	if !strings.HasPrefix(channelUsername, "@") {
		channelUsername = "@" + channelUsername
	}

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		cm, err := api.GetChatMember(channelUsername, userID)
		if err == nil {
			// Successfully got response
			if cm.Status == "left" || cm.Status == "kicked" {
				return false
			}
			return true
		}

		// If error indicates user is not a member at all, or we lack permission to view,
		// treat as "not a member" immediately.
		if strings.Contains(err.Error(), "no such group or user") ||
			strings.Contains(err.Error(), "permission_denied") {
			return false
		}

		// Other errors (network, server) – log and retry
		logger.Error.Printf("GetChatMember attempt %d failed for %s: %v", attempt+1, channelUsername, err)
		if attempt < maxRetries-1 {
			time.Sleep(1 * time.Second)
		}
	}
	// All retries exhausted – assume member to avoid false deactivation
	logger.Error.Printf("All retries failed for %s, assuming user is member", channelUsername)
	return true
}

func IsUserInChannels(userID int) bool {
	channelUsernames := []string{config.Channel1, config.Channel2, config.Channel3}
	for _, ch := range channelUsernames {
		if ch == "" {
			continue
		}
		if !isUserInChannelWithRetry(ch, userID) {
			return false
		}
	}
	return true
}
