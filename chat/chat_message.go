package chat

type ChatMessage struct {
	UserId  uint32 `json:"userId"`
	RoomId  uint32 `json:"roomId"`
	Content string `json:"content"`
	Type    string `json:"type"`
}
