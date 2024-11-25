package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

var (
	serverHost = flag.String("h", "localhost", "server host")
	serverPort = flag.Int("p", 8972, "server port")
)

func main() {
	flag.Parse()

	// 连接服务器
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *serverHost, *serverPort))
	if err != nil {
		fmt.Println("failed to connect to server:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("failed to connect to chat server")

	// 启动goroutine接收服务器消息
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("\ndisconnected from server")
				os.Exit(0)
			}
			fmt.Print(string(buffer[:n]))
		}
	}()

	// 从标准输入读取消息并发送
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := scanner.Text()
		msg = strings.TrimSpace(msg)

		if msg == "" {
			continue
		}

		// 发送消息到服务器
		_, err := conn.Write([]byte(msg + "\n"))
		if err != nil {
			fmt.Println("failed to send message:", err)
			break
		}

		// 如果输入 /quit 则退出
		if msg == "/quit" {
			fmt.Println("goodbye!")
			return
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("failed to read input:", err)
	}
}
