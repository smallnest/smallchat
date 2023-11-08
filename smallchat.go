package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	maxClients = 1000
	maxNickLen = 32
)

var (
	serverPort = flag.Int("p", 8972, "server port")
)

type Client struct {
	conn     net.Conn
	nick     string
	readChan chan string
}

func (c Client) startRecv() {
	for msg := range c.readChan {
		c.conn.Write([]byte(msg))
	}
}

type ChatState struct {
	listener net.Listener

	clientsLock sync.RWMutex
	clients     map[net.Conn]*Client
	numClients  int
}

var chatState = &ChatState{
	clients: make(map[net.Conn]*Client),
}

func initChat() {
	var err error
	chatState.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", *serverPort))
	if err != nil {
		fmt.Println("listen error:", err)
		os.Exit(1)
	}
}

func handleClient(client *Client) {
	// 发送欢迎信息
	welcomeMsg := "Welcome Simple Chat! Use /nick to change nick name.\n"
	client.conn.Write([]byte(welcomeMsg))

	buf := make([]byte, 256)
	for {
		n, err := client.conn.Read(buf)
		if err != nil {
			fmt.Printf("client left: %s\n", client.conn.RemoteAddr())
			chatState.clientsLock.Lock()
			close(client.readChan)
			client.conn.Close()
			delete(chatState.clients, client.conn)
			chatState.numClients--
			chatState.clientsLock.Unlock()
			return
		}

		msg := string(buf[:n])
		msg = strings.TrimSpace(msg)
		if len(msg) > 0 && msg[0] == '/' {
			// 处理命令
			parts := strings.SplitN(msg, " ", 2)
			cmd := parts[0]
			if cmd == "/nick" && len(parts) > 1 {
				if len(parts[1]) > maxNickLen {
					client.conn.Write([]byte("nick name too long\n"))
					continue
				}
				client.nick = parts[1]
			}
			continue
		}

		fmt.Printf("%s: %s\n", client.nick, msg)

		// 将消息转发给其他客户端
		chatState.clientsLock.RLock()
		for _, cl := range chatState.clients {
			if cl != client {
				cl.readChan <- client.nick + ": " + msg + "\n"
			}
		}
		chatState.clientsLock.RUnlock()
	}
}

func main() {
	flag.Parse()

	initChat()

	for {
		conn, err := chatState.listener.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}

		client := &Client{conn: conn}
		client.nick = fmt.Sprintf("user%d", conn.RemoteAddr().(*net.TCPAddr).Port)
		client.readChan = make(chan string, 5)

		chatState.clientsLock.Lock()
		if chatState.numClients >= maxClients {
			fmt.Printf("too many clients, reject %s\n", conn.RemoteAddr())
			conn.Close()
			chatState.clientsLock.Unlock()
			continue
		}
		chatState.clients[conn] = client
		chatState.numClients++
		chatState.clientsLock.Unlock()

		go handleClient(client)
		go client.startRecv()
		fmt.Printf("new client: %s\n", conn.RemoteAddr())
	}
}
