package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var (
	clients   = make(map[*Client]bool) // 存储所有连接的客户端
	broadcast = make(chan []byte)      // 广播消息通道
	upgrader  = websocket.Upgrader{}   // 用于升级 HTTP 连接为 WebSocket 连接的 Upgrader
)

// func homePage(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Home HTTP")
// }

// func setupRoutes() {
// 	http.HandleFunc("/", homePage)
// 	http.HandleFunc("/ws", wsEndpoint)
// }

// var upgrader = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// 	CheckOrigin:     func(r *http.Request) bool { return true },
// }

// func reader(conn *websocket.Conn) {
// 	for {
// 		// read in a message
// 		messageType, p, err := conn.ReadMessage()
// 		if err != nil {
// 			log.Println(err)
// 			return
// 		}
// 		// print out that message for clarity
// 		log.Println(string(p))
// 		s := "the word is sent"
// 		// 這個 WriteMessage 會傳送
// 		if err := conn.WriteMessage(messageType, []byte(s)); err != nil {
// 			log.Println(err)
// 			return
// 		}
// 	}
// }

// func wsEndpoint(w http.ResponseWriter, r *http.Request) {
// 	// upgrade this connection to a WebSocket
// 	// c
// 	ws, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println(err)
// 	}

// 	log.Println("Client Connected")
// 	err = ws.WriteMessage(1, []byte("Hi Client!"))
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	// listen indefinitely for new messages coming
// 	// through on our WebSocket connection
// 	reader(ws)
// }

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 将 HTTP 连接升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	// 创建新的客户端对象
	client := &Client{
		conn: conn,
		send: make(chan []byte),
	}

	// 将新连接的客户端添加到 clients 映射中
	clients[client] = true

	go client.writePump()
	go client.readPump()
	// 调用 broadcastMessages 函数来处理消息广播
	broadcastMessages()
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		log.Printf("here is message:%+v", string(message))
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("Error: %v", err)
			}
			break
		}
		broadcast <- message
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message := <-c.send:
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}
		}
	}
}

func broadcastMessages() {
	for {
		message := <-broadcast

		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

func main() {
	// 创建静态文件服务器来提供客户端页面
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)

	// 定义 WebSocket 路由
	http.HandleFunc("/ws", handleWebSocket)

	// 启动服务器
	log.Println("Server started at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
