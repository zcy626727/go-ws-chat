package chat

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	// 写超时时间
	writeWait = 10 * time.Second

	// 读超时时间
	pongWait = 10 * time.Second

	// ping 计时器 间隔时间，这个值必须比writeWait和pongWait小
	pingPeriod = (pongWait * 7) / 10

	// 对等方允许的最大消息大小。
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 解决跨域问题
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 是 websocket 连接和 ChatRoom 之间的中间人。
type Client struct {
	room *ChatRoom

	conn *websocket.Conn
}

// readPump
// 读取客户端消息并将其发送到MQ
// 读取连接请求将其加入到RoomList
func (c *Client) readPump() {
	println("go readPump")

	c.conn.SetReadLimit(maxMessageSize)
	// 设置超时时间
	c.conn.SetReadDeadline(time.Now().Add(pongWait * 2))
	// 客户端返回pong后延长时间
	c.conn.SetPongHandler(func(string) error {
		println("收到pong，刷新超时时间")
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		println("readPump")
		// 读取消息
		_, message, err := c.conn.ReadMessage()
		var msgJson ChatMessage
		// 消息解析
		err = json.Unmarshal(message, &msgJson)
		if err != nil {
			println("遇到错误退出读循环")
			return
		}
		switch msgJson.Type {
		case "join":
			// 将客户端注册到room
			if room, ok := ChatRoomMap[msgJson.RoomId]; ok {
				// 房间已存在
				room.clients[c] = true
				room.roomId = msgJson.RoomId
			} else {
				// 房间不存在，创建房间
				var cr = ChatRoom{
					roomId: msgJson.RoomId,
					clients: map[*Client]bool{
						c: true,
					},
				}

				// 添加到map
				ChatRoomMap[msgJson.RoomId] = cr
			}
			println("join")
		case "exit":
			if room, ok := ChatRoomMap[msgJson.RoomId]; ok {
				// 从房间删除客户端
				delete(room.clients, c)
			}
		case "message":
			// 将消息发送到MQ
			pushMessageToMQ(message)
		}

	}
}

// writePump 监听MQ消息并将其发送到客户端
func (c *Client) writePump() {
	println("go writePump")

	// 返回一个计时器，每隔一段时间向自身通道发送一个值
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	// 返回一个管道，用于监听消息
	consume, err := MQChannel.Consume(
		MQQueue.Name, // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Fatalf("Error consuming: %s", err)
		return
	}
	for {
		println("writePump")
		select {
		// 监听MQ，将消息发送到客户端
		case c := <-consume:
			message := c.Body
			var msgJson ChatMessage
			// 消息解析
			err = json.Unmarshal(message, &msgJson)
			if err != nil {
				return
			}
			// 根据roomId查询到room，然后将消息发送到client列表
			room := ChatRoomMap[msgJson.RoomId]

			for client := range room.clients {
				// 遍历客户端，然后将消息发送出去
				err := client.conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					return
				}
			}
		//定期写入 ping 消息，检测连线是否正常，客户端会返回pong
		case <-ticker.C:
			println("发送ping")
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// 如果写入失败就结束
				return
			}
		}
	}
}

// ServeWs 处理来自 peer 的 WebSocket 请求。
func ServeWs(w http.ResponseWriter, r *http.Request) {

	// 将 http 升级为 websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// 创建client，保存连接
	client := &Client{room: nil, conn: conn}

	// 开启两个协程，分别处理当前客户端连接的写入和写出操作
	go client.writePump()
	go client.readPump()
}
