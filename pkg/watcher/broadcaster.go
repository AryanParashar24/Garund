package watcher

import (
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients = make(map[*websocket.Conn]bool)
	mutex   sync.Mutex
)

func AddClient(conn *websocket.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	clients[conn] = true
}

func RemoveClient(conn *websocket.Conn) {
	mutex.Lock()
	defer mutex.Unlock()

	delete(clients, conn)
	conn.Close()
}

func Broadcast(v any) {

	mutex.Lock()
	defer mutex.Unlock()

	for conn := range clients {

		if err := conn.WriteJSON(v); err != nil {

			conn.Close()

			delete(clients, conn)

		}

	}

}
