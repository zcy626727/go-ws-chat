//The application runs one goroutine for the ChatRoom and two goroutines for each Client. The goroutines communicate with each other using channels. The ChatRoom has channels for registering clients, unregistering clients and broadcasting messages. A Client has a buffered channel of outbound messages. One of the client's goroutines reads messages from this channel and writes the messages to the websocket. The other client goroutine reads messages from the websocket and sends them to the room.

package main

import (
	"flag"
	"log"
	"net/http"
	"ws/chat"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()

	http.HandleFunc("/", serveHome)
	// 处理客户端连接请求，启动两个协程，一个用于接收客户端消息并转发到MQ、一个用于监听MQ并转发到客户端
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		chat.ServeWs(w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
