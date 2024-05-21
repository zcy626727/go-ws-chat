package chat

// ChatRoom 维护一组活动 client，并将消息广播到 clients
type ChatRoom struct {
	roomId uint32
	// 该聊天室的客户端连接列表
	clients map[*Client]bool
}

var ChatRoomMap = make(map[uint32]ChatRoom)
