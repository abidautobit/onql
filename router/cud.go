package router

import (
	"encoding/json"
	"fmt"
	"onql/database"
	"onql/dsl"
)

type insertData struct {
	DB      string         `json:"db"`
	Table   string         `json:"table"`
	Records map[string]any `json:"records"`
}

type updateData struct {
	DB        string         `json:"db"`
	Table     string         `json:"table"`
	Records   map[string]any `json:"records"`
	Query     string         `json:"query"`
	Ids       []string       `json:"ids"`
	Protopass string         `json:"protopass"`
}

type deleteData struct {
	DB        string   `json:"db"`
	Table     string   `json:"table"`
	Query     string   `json:"query"`
	Ids       []string `json:"ids"`
	Protopass string   `json:"protopass"`
}

// insert with records
func HandleInsert(msg *Message) {
	data := msg.Payload
	insData := insertData{}

	if err := json.Unmarshal([]byte(data), &insData); err != nil {
		fmt.Println(err)
		// fmt.Println(data)
		// Handle error
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		rMsg := &Message{
			ID:      "cud",
			RID:     msg.RID,
			Target:  msg.ID,
			Payload: string(payload),
			Type:    "response",
		}
		Route(rMsg)
		return
	}
	// Call the insert function
	pk, err := database.Insert(insData.DB, insData.Table, insData.Records)
	payload := ""
	payloadError := ""
	if err != nil {
		payloadError = err.Error()
	} else {
		payload = pk
	}
	response, _ := json.Marshal(map[string]string{"data": payload, "error": payloadError})

	rMsg := &Message{
		ID:      "cud",
		RID:     msg.RID,
		Target:  msg.ID,
		Payload: string(response),
		Type:    "response",
	}
	Route(rMsg)
	nats.Publish("react.database", []byte(fmt.Sprintf(`{"db":"%s","table":"%s","action":"insert","id":["%s"]}`, insData.DB, insData.Table, payload)))
}

// update with query
func HandleUpdate(msg *Message) {
	data := msg.Payload
	updData := updateData{}

	if err := json.Unmarshal([]byte(data), &updData); err != nil {
		fmt.Println(err)
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		// Handle error
		rMsg := &Message{
			ID:      "cud",
			RID:     msg.RID,
			Target:  msg.ID,
			Payload: string(payload),
			Type:    "response",
		}
		Route(rMsg)
		return
	}

	pks := []string{}

	if updData.Query != "" {
		result, err := dsl.Execute(updData.Protopass, updData.Query, "", []string{})
		if err != nil {
			payload, _ := json.Marshal(map[string]string{"error": err.Error()})
			rMsg := &Message{
				ID:      "cud",
				RID:     msg.RID,
				Target:  msg.ID,
				Payload: string(payload),
				Type:    "response",
			}
			Route(rMsg)
			return
		}
		var ok bool
		pks, ok = result.([]string)
		if !ok {
			payload, _ := json.Marshal(map[string]string{"error": "idies not returned by query"})

			rMsg := &Message{
				ID:      "cud",
				RID:     msg.RID,
				Target:  msg.ID,
				Payload: string(payload),
				Type:    "response",
			}
			Route(rMsg)
			return
		}
	}

	if len(updData.Ids) != 0 {
		// err := json.Unmarshal([]byte(updData.Ids), &pks)
		// if err != nil {
		// 	payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		// 	rMsg := &Message{
		// 		ID:      "cud",
		// 		RID:     msg.RID,
		// 		Target:  msg.ID,
		// 		Payload: string(payload),
		// 		Type:    "response",
		// 	}
		// 	Route(rMsg)
		// 	return
		// }
		pks = updData.Ids
	}

	payload := ""
	payloadError := ""

	for _, pk := range pks {
		// Call the update function
		updData.Records["id"] = pk
		err := database.UpdatePartial(updData.DB, updData.Table, updData.Records)
		if err != nil {
			payloadError = err.Error()
			break
		} else {
			payload = "success"
		}
	}

	if len(pks) == 0 {
		payload = "success"
	}

	response, _ := json.Marshal(map[string]string{"error": payloadError, "data": payload})

	rMsg := &Message{
		ID:      "cud",
		RID:     msg.RID,
		Target:  msg.ID,
		Payload: string(response),
		Type:    "response",
	}
	Route(rMsg)
	nats.Publish("react.database", []byte(fmt.Sprintf(`{"db":"%s","table":"%s","action":"update","id":%v}`, updData.DB, updData.Table, pks)))
}

func HandleDelete(msg *Message) {
	data := msg.Payload
	delData := deleteData{}

	if err := json.Unmarshal([]byte(data), &delData); err != nil {
		fmt.Println(err)
		// Handle error
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		rMsg := &Message{
			ID:      "cud",
			RID:     msg.RID,
			Target:  msg.ID,
			Payload: string(payload),
			Type:    "response",
		}
		Route(rMsg)
		return
	}

	pks := []string{}

	if delData.Query != "" {
		result, err := dsl.Execute(delData.Protopass, delData.Query, "", []string{})
		if err != nil {
			payload, _ := json.Marshal(map[string]string{"error": err.Error()})
			rMsg := &Message{
				ID:      "cud",
				RID:     msg.RID,
				Target:  msg.ID,
				Payload: string(payload),
				Type:    "response",
			}
			Route(rMsg)
			return
		}
		var ok bool
		pks, ok = result.([]string)
		if !ok {
			payload, _ := json.Marshal(map[string]string{"error": "idies not returned by query"})
			rMsg := &Message{
				ID:      "cud",
				RID:     msg.RID,
				Target:  msg.ID,
				Payload: string(payload),
				Type:    "response",
			}
			Route(rMsg)
			return
		}
	}

	if len(delData.Ids) != 0 {
		// err := json.Unmarshal([]byte(delData.Ids), &pks)
		// if err != nil {
		// 	payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		// 	rMsg := &Message{
		// 		ID:      "cud",
		// 		RID:     msg.RID,
		// 		Target:  msg.ID,
		// 		Payload: string(payload),
		// 		Type:    "response",
		// 	}
		// 	Route(rMsg)
		// 	return
		// }
		pks = delData.Ids
	}

	payload := ""
	payloadError := ""

	// Call the delete function
	if len(pks) != 0 {
		err := database.Delete(delData.DB, delData.Table, pks)
		if err != nil {
			payloadError = err.Error()
		} else {
			payload = "success"
		}
	} else {
		payload = "success"
	}

	response, _ := json.Marshal(map[string]string{"error": payloadError, "data": payload})

	rMsg := &Message{
		ID:      "cud",
		RID:     msg.RID,
		Target:  msg.ID,
		Payload: string(response),
		Type:    "response",
	}
	Route(rMsg)
	nats.Publish("react.database", []byte(fmt.Sprintf(`{"db":"%s","table":"%s","action":"delete","id":%v}`, delData.DB, delData.Table, pks)))
}
