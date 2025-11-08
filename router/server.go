package router

import "strings"

const msgDelimeter = "\x1E"

var connQueryMap = make(map[string]*func(string))

func HandleServerRequest(query string, connUniqueId string, responder *func(string)) {
	parts := strings.Split(query, msgDelimeter)
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
	response := msg.RID + msgDelimeter + msg.ID + msgDelimeter + string(msg.Payload)
	(*connQueryMap[rid])(response)
	delete(connQueryMap, rid)
}
