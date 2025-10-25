package main

import (
	"fmt"
	"net"
	"os"
)

const addr = ":6969"

type Client struct {
	Username string
	Addr     string
	Conn     *net.Conn
}

var clients map[string]*Client

type MessageKind int

const (
	ClientConnected MessageKind = iota
	ClientNewMessage
	ClientDisconnected
)

type Message struct {
	Kind   MessageKind
	Client *Client
	Msg    []byte // TODO: move to a buffer instead of a slice
}

func main() {
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not listen to address %s: %s\n", addr, err)
		os.Exit(1)
	}

	fmt.Printf("Listening to address: %s\n", addr)

	clients = make(map[string]*Client)

	messagesCh := make(chan Message)
	go handleServer(messagesCh)

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		_, err = conn.Write([]byte("Hello, choose an username:\r\n"))
		if err != nil {
			continue
		}

		b := make([]byte, 16)
		// TODO: handle duplicates
		n, err := conn.Read(b)
		if err != nil {
			continue
		}

		client := &Client{
			Username: string(b[:n-2]),
			Addr:     conn.RemoteAddr().String(),
			Conn:     &conn,
		}

		messagesCh <- Message{
			Kind:   ClientConnected,
			Client: client,
		}

		go handleClientConnection(client, messagesCh)
	}
}

func handleServer(messagesCh chan Message) {
	for {
		message := <-messagesCh
		username := message.Client.Username

		switch message.Kind {
		case ClientConnected:
			clients[username] = message.Client

			for _, client := range clients {
				conn := *(client.Conn)
                msg := fmt.Sprintf("%s joined the chat\r\n", username)
				conn.Write([]byte(msg))
			}

		case ClientNewMessage:
			for a, client := range clients {
				if a == username {
					continue
				}
				conn := *(client.Conn)
                msg := fmt.Sprintf("%s: %s\r\n", username, message.Msg)
				conn.Write([]byte(msg))
			}
		case ClientDisconnected:
			delete(clients, username)
            for _, client := range clients {
                conn := *(client.Conn)
                msg := fmt.Sprintf("%s has left the chat\r\n", username)
                conn.Write([]byte(msg))
            }
		}
	}
}

func handleClientConnection(client *Client, messagesCh chan Message) {
	buffer := make([]byte, 256)
	conn := *(client.Conn)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not read from client: %s. Closing connection..\n", err)
			conn.Close()
			messagesCh <- Message{
				Kind:   ClientDisconnected,
				Client: client,
			}
			return
		}

		messagesCh <- Message{
			Kind:   ClientNewMessage,
			Client: client,
			Msg:    buffer[:n-2],
		}
	}

}
