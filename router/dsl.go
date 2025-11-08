package router

import (
	"encoding/json"
	"fmt"
	"onql/dsl"
	"strings"
)

func HandleDslRequest(msg *Message) {
	var payload string
	var payloadError string
	var data map[string]string

	json.Unmarshal([]byte(msg.Payload), &data)
	values := strings.Split(data["ctxvalues"], ",")

	fmt.Println(data)

	result, err := dsl.Execute(data["protopass"], data["query"], data["ctxkey"], values)
	if err != nil {
		payloadError = err.Error()
	} else {
		json, err := json.Marshal(result)
		if err != nil {
			payloadError = err.Error()
		} else {
			payload = string(json)
		}
	}

	response, _ := json.Marshal(map[string]string{"data": payload, "error": payloadError})
	Route(&Message{
		RID:     msg.RID,
		ID:      "onql",
		Target:  msg.ID,
		Payload: string(response),
		Type:    "response",
	})
}
