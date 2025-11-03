// Responsible for transfer data to appropriate programs
// (server,extension,database) with appropriate channels.
package router

func Route(msg *Message) {
	switch msg.Type {
	case "request":
		handleRequest(msg)
	case "response":
		handleResponse(msg)
	}
}

func handleRequest(msg *Message) {
	switch msg.Target {
	case "database":
		handleDatabaseRequest(msg)
	case "onql":
		HandleDslRequest(msg)
	case "insert":
		HandleInsert(msg)
	case "update":
		HandleUpdate(msg)
	case "delete":
		HandleDelete(msg)
	case "subscribe":
		handleStreamRequest(msg)
	default:
		handleExtensionMessage(msg)
	}
}

func handleResponse(msg *Message) {
	switch msg.Target {
	case "server":
		handleServerResponse(msg)
	default:
		handleExtensionMessage(msg)
	}
}
