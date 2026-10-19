package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dalbot/config"
	"dalbot/logger"
)

const base = "https://tapi.bale.ai/bot"

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

type Response struct {
	Ok          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

type User struct {
	ID        int    `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Title      string `json:"title,omitempty"`
	Username   string `json:"username,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty"`
	InviteLink string `json:"invite_link,omitempty"`
}

type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

type Video struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

type Audio struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Duration     int        `json:"duration"`
	Title        string     `json:"title,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

type Sticker struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	IsAnimated   bool       `json:"is_animated"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	Emoji        string     `json:"emoji,omitempty"`
	SetName      string     `json:"set_name,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

type Animation struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Duration     int        `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

type Document struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     string     `json:"file_name,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	FileSize     int        `json:"file_size,omitempty"`
}

type PollOption struct {
	Text       string `json:"text"`
	VoterCount int    `json:"voter_count"`
}

type Poll struct {
	ID       string       `json:"id"`
	Question string       `json:"question"`
	Options  []PollOption `json:"options"`
	IsClosed bool         `json:"is_closed"`
}

type Message struct {
	MessageID int      `json:"message_id"`
	From      *User    `json:"from,omitempty"`
	Date      int      `json:"date"`
	Chat      Chat     `json:"chat"`
	Text      string   `json:"text,omitempty"`
	ReplyTo   *Message `json:"reply_to_message,omitempty"`

	Photo     []PhotoSize `json:"photo,omitempty"`
	Video     *Video      `json:"video,omitempty"`
	Audio     *Audio      `json:"audio,omitempty"`
	Voice     *Voice      `json:"voice,omitempty"`
	Sticker   *Sticker    `json:"sticker,omitempty"`
	Animation *Animation  `json:"animation,omitempty"`
	Document  *Document   `json:"document,omitempty"`

	ForwardFrom     *User `json:"forward_from,omitempty"`
	ForwardFromChat *Chat `json:"forward_from_chat,omitempty"`
	ForwardDate     int   `json:"forward_date,omitempty"`

	NewChatMembers []User `json:"new_chat_members,omitempty"`
	LeftChatMember *User  `json:"left_chat_member,omitempty"`

	PinnedMessage *Message `json:"pinned_message,omitempty"`
	Poll          *Poll    `json:"poll,omitempty"`

	Caption         string                `json:"caption,omitempty"`
	CaptionEntities []interface{}         `json:"caption_entities,omitempty"`
	Entities        []interface{}         `json:"entities,omitempty"`
	ReplyMarkup     *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
	Poll          *Poll          `json:"poll,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type ChatMember struct {
	User   User   `json:"user"`
	Status string `json:"status"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SendMessageRequest struct {
	ChatID       string                 `json:"chat_id"`
	Text         string                 `json:"text"`
	ReplyToMsgID int                    `json:"reply_to_message_id,omitempty"`
	ReplyMarkup  *InlineKeyboardMarkup  `json:"reply_markup,omitempty"`
}

type AnswerCallbackRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

type BanChatMemberRequest struct {
	ChatID string `json:"chat_id"`
	UserID int    `json:"user_id"`
}

type EditMessageTextRequest struct {
	ChatID      string                `json:"chat_id"`
	MessageID   int                   `json:"message_id"`
	Text        string                `json:"text"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type ChatPermissions struct {
	CanSendMessages       bool `json:"can_send_messages"`
	CanSendMediaMessages  bool `json:"can_send_media_messages"`
	CanSendPolls          bool `json:"can_send_polls"`
	CanSendOtherMessages  bool `json:"can_send_other_messages"`
	CanAddWebPagePreviews bool `json:"can_add_web_page_previews"`
	CanChangeInfo         bool `json:"can_change_info"`
	CanInviteUsers        bool `json:"can_invite_users"`
	CanPinMessages        bool `json:"can_pin_messages"`
}

type RestrictChatMemberRequest struct {
	ChatID      string          `json:"chat_id"`
	UserID      int             `json:"user_id"`
	Permissions ChatPermissions `json:"permissions"`
}

// doRequest sends a POST form request and returns the parsed Response.
func doRequest(method string, params map[string]string) (*Response, error) {
	fullURL := config.BaseURL + "/" + method
	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	resp, err := httpClient.PostForm(fullURL, data)
	if err != nil {
		logger.Error.Printf("HTTP request failed (%s): %v", method, err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error.Printf("Failed to read response body (%s): %v", method, err)
		return nil, err
	}

	// اگر پاسخ با '{' شروع نشود (احتمالاً HTML یا خطای وب‌سرور)، بازگرداندن خطای شفاف
	if len(body) > 0 && body[0] != '{' {
		logger.Info.Printf("Bale returned non-JSON for %s: %s", method, string(body[:min(len(body), 100)]))
		return nil, fmt.Errorf("bale returned non-JSON (maintenance?)")
	}

	var r Response
	if err := json.Unmarshal(body, &r); err != nil {
		logger.Error.Printf("Failed to decode JSON (%s): %v", method, err)
		return nil, err
	}
	if !r.Ok {
		return &r, fmt.Errorf("API error %d: %s", r.ErrorCode, r.Description)
	}
	return &r, nil
}

// doPostJSON sends a JSON POST request and returns the parsed Response.
func doPostJSON(method string, payload interface{}) (*Response, error) {
	fullURL := config.BaseURL + "/" + method
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Post(fullURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		logger.Error.Printf("HTTP POST failed (%s): %v", method, err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(respBody) > 0 && respBody[0] != '{' {
		logger.Info.Printf("Bale returned non-JSON for %s: %s", method, string(respBody[:min(len(respBody), 100)]))
		return nil, fmt.Errorf("bale returned non-JSON (maintenance?)")
	}

	var r Response
	if err := json.Unmarshal(respBody, &r); err != nil {
		return nil, err
	}
	if !r.Ok {
		return &r, fmt.Errorf("API error %d: %s", r.ErrorCode, r.Description)
	}
	return &r, nil
}

func SendMessage(chatID string, text string) (*Message, error) {
	req := SendMessageRequest{ChatID: chatID, Text: text}
	resp, err := doPostJSON("sendMessage", req)
	if err != nil {
		return nil, err
	}
	var msg Message
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func SendMessageWithKeyboard(chatID string, text string, keyboard InlineKeyboardMarkup) (*Message, error) {
	req := SendMessageRequest{ChatID: chatID, Text: text, ReplyMarkup: &keyboard}
	resp, err := doPostJSON("sendMessage", req)
	if err != nil {
		return nil, err
	}
	var msg Message
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func SendReply(chatID string, replyTo int, text string) (*Message, error) {
	req := SendMessageRequest{ChatID: chatID, Text: text, ReplyToMsgID: replyTo}
	resp, err := doPostJSON("sendMessage", req)
	if err != nil {
		return nil, err
	}
	var msg Message
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func DeleteMessage(chatID string, messageID int) error {
	params := map[string]string{
		"chat_id":    chatID,
		"message_id": strconv.Itoa(messageID),
	}
	_, err := doRequest("deleteMessage", params)
	return err
}

func GetUpdates(offset int) ([]Update, error) {
	params := map[string]string{
		"offset":  strconv.Itoa(offset),
		"timeout": "5",
	}
	resp, err := doRequest("getUpdates", params)
	if err != nil {
		return nil, err
	}
	var updates []Update
	if err := json.Unmarshal(resp.Result, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func GetChatMember(chatID string, userID int) (*ChatMember, error) {
	params := map[string]string{
		"chat_id": chatID,
		"user_id": strconv.Itoa(userID),
	}
	resp, err := doRequest("getChatMember", params)
	if err != nil {
		return nil, err
	}
	var cm ChatMember
	if err := json.Unmarshal(resp.Result, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

func BanChatMember(chatID string, userID int) error {
	req := BanChatMemberRequest{ChatID: chatID, UserID: userID}
	_, err := doPostJSON("banChatMember", req)
	return err
}

func UnbanChatMember(chatID string, userID int) error {
	req := BanChatMemberRequest{ChatID: chatID, UserID: userID}
	_, err := doPostJSON("unbanChatMember", req)
	return err
}

func RestrictChatMember(chatID string, userID int, canSend bool) error {
	perms := ChatPermissions{
		CanSendMessages:       canSend,
		CanSendMediaMessages:  canSend,
		CanSendPolls:          canSend,
		CanSendOtherMessages:  canSend,
		CanAddWebPagePreviews: canSend,
	}
	req := RestrictChatMemberRequest{ChatID: chatID, UserID: userID, Permissions: perms}
	_, err := doPostJSON("restrictChatMember", req)
	return err
}

func UnrestrictChatMember(chatID string, userID int) error {
	perms := ChatPermissions{
		CanSendMessages:       true,
		CanSendMediaMessages:  true,
		CanSendPolls:          true,
		CanSendOtherMessages:  true,
		CanAddWebPagePreviews: true,
	}
	req := RestrictChatMemberRequest{ChatID: chatID, UserID: userID, Permissions: perms}
	_, err := doPostJSON("restrictChatMember", req)
	return err
}

func ExportChatInviteLink(chatID string) (string, error) {
	params := map[string]string{
		"chat_id": chatID,
	}
	resp, err := doRequest("exportChatInviteLink", params)
	if err != nil {
		return "", err
	}
	var link string
	if err := json.Unmarshal(resp.Result, &link); err != nil {
		var obj struct {
			InviteLink string `json:"invite_link"`
		}
		if err2 := json.Unmarshal(resp.Result, &obj); err2 != nil {
			return "", fmt.Errorf("failed to parse invite link: %v (raw: %s)", err2, string(resp.Result))
		}
		link = obj.InviteLink
	}
	return link, nil
}

func AnswerCallback(queryID string, text string, showAlert bool) error {
	req := AnswerCallbackRequest{CallbackQueryID: queryID, Text: text, ShowAlert: showAlert}
	_, err := doPostJSON("answerCallbackQuery", req)
	return err
}

func GetChat(chatID string) (*Chat, error) {
	params := map[string]string{
		"chat_id": chatID,
	}
	resp, err := doRequest("getChat", params)
	if err != nil {
		return nil, err
	}
	var chat Chat
	if err := json.Unmarshal(resp.Result, &chat); err != nil {
		return nil, err
	}
	return &chat, nil
}

func SendSticker(chatID string, fileID string) error {
	params := map[string]string{
		"chat_id": chatID,
		"sticker": fileID,
	}
	_, err := doRequest("sendSticker", params)
	return err
}

func EditMessageText(chatID string, messageID int, text string, keyboard *InlineKeyboardMarkup) error {
	req := EditMessageTextRequest{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
	}
	if keyboard != nil {
		req.ReplyMarkup = keyboard
	}
	_, err := doPostJSON("editMessageText", req)
	return err
}

func RevokeChatInviteLink(chatID, inviteLink string) error {
	params := map[string]string{
		"chat_id":     chatID,
		"invite_link": inviteLink,
	}
	_, err := doRequest("revokeChatInviteLink", params)
	return err
}

func PinChatMessage(chatID string, messageID int) error {
	params := map[string]string{
		"chat_id":    chatID,
		"message_id": strconv.Itoa(messageID),
	}
	_, err := doRequest("pinChatMessage", params)
	return err
}

func UnpinChatMessage(chatID string, messageID int) error {
	params := map[string]string{
		"chat_id": chatID,
	}
	if messageID != 0 {
		params["message_id"] = strconv.Itoa(messageID)
	}
	_, err := doRequest("unpinChatMessage", params)
	return err
}

func FixInviteLink(link string) string {
	if !strings.HasPrefix(link, "http") {
		if strings.HasPrefix(link, "ble.ir/") {
			return "https://" + link
		} else {
			return "https://ble.ir/" + strings.TrimLeft(link, "/")
		}
	}
	return link
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
