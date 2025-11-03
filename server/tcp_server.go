package server

import (
	"bufio"
	"log"
	"net"
	"onql/config"
	"onql/router"
	"strings"

	"github.com/google/uuid"
)

func Setup() {
	port := config.Env("SERVER_PORT")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Error starting TCP server:", err)
	}

	defer listener.Close()
	log.Println("Server started on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	connUniqueId := uuid.NewString()

	handleResponse := func(response string) {
		response += "\x04" // Append end-of-message character
		if _, err := conn.Write([]byte(response)); err != nil {
			log.Println("Write failed:", err)
		}
	}

	for {
		// This is uncommon character for end of message
		// it is used to avoid conflicts with user input.
		message, err := reader.ReadString('\x04') // Read line-delimited messages
		if err != nil {
			log.Println("Connection closed:", err)
			return
		}

		query := strings.TrimSuffix(message, "\x04")
		log.Printf("Received: %s", query)

		// Pass query to router with responder function
		go router.HandleServerRequest(query, connUniqueId, &handleResponse)
	}
}
