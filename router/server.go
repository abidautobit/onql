package router

import "strings"

var connQueryMap = make(map[string]*func(string))

func HandleServerRequest(query string, connUniqueId string, responder *func(string)) {
	parts := strings.Split(query, "\x1E")
	msg := &Message{
		ID:      "server",
		RID:     parts[0],
		Target:  parts[1],
		Payload: parts[2],
		Type:    "request",
	}
	connQueryMap[msg.RID] = responder
	Route(msg)
}

// Responsible for taking queries from tcp server and response them.
func handleServerResponse(msg *Message) {
	rid := msg.RID
	response := msg.RID + "\x1E" + msg.ID + "\x1E" + string(msg.Payload)
	(*connQueryMap[rid])(response)
	delete(connQueryMap, rid)
}
