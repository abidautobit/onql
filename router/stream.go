package router

import (
	"encoding/json"
	"log"
	"onql/dsl"
	"onql/dsl/evaluator"
	"onql/dsl/parser"
	"onql/engine"
	"reflect"
	"sync"
)

// id => queries => connections
// type StreamMap map[string]map[string]map[string]bool
// ids => queries => contextkey => values => connections
type QueryConnMap struct {
	Ctxkey    string
	Ctxvalues []string
	Conns     map[string]bool
	Eval      *evaluator.Evaluator
}
type StreamMap map[string]map[string]map[string]*QueryConnMap

// ids => queries => evaluator{} => conns

// global store + mutex
var (
	streams   = make(StreamMap)
	streamsMu sync.RWMutex
)

func StartStreamer() {
	nats.Subscribe("database.react", handleDatabaseReaction)
	log.Println("Streamer started")
}

func handleDatabaseReaction(msg *engine.Msg) {
	// id is now an array of strings
	var data struct {
		ID []string `json:"id"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("⚠️  could not unmarshal database react message via nats: %v", err)
		return
	}

	// fetch streaming queries and connections with ids
	for _, id := range data.ID {
		streamsMu.RLock()
		proto := streams[id]
		streamsMu.RUnlock()

		for protopass, queries := range proto {
			_ = protopass // use as needed
			for query, connsMap := range queries {
				_ = query // use as needed
				err := connsMap.Eval.Eval()
				if err != nil {
					//do something
					continue
				}
				//stream to all
				Stream(connsMap.Eval.Result, connsMap.Conns)
			}
		}
	}
}

type request struct {
	Onquery   string   `json:"onquery"`
	Query     string   `json:"query"`
	Protopass string   `json:"protopass"`
	Ctxkey    string   `json:"ctxkey"`
	Ctxvalues []string `json:"ctxvalues"`
}

func handleStreamRequest(msg *Message) {
	var pks []string
	var data request
	if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
		Route(&Message{
			RID:     msg.RID,
			Target:  msg.ID,
			ID:      "subscribe",
			Payload: err.Error(),
			Type:    "response",
		})
		return
	}

	if data.Onquery != "" {
		result, err := dsl.Execute(data.Protopass, data.Onquery, data.Ctxkey, data.Ctxvalues)
		if err != nil {
			rMsg := &Message{
				ID:      "subscribe",
				RID:     msg.RID,
				Target:  msg.ID,
				Payload: err.Error(),
				Type:    "response",
			}
			Route(rMsg)
			return
		}
		var ok bool
		pks, ok = result.([]string)
		if !ok {
			rMsg := &Message{
				ID:      "subscribe",
				RID:     msg.RID,
				Target:  msg.ID,
				Payload: "invalid query result",
				Type:    "response",
			}
			Route(rMsg)
			return
		}
	}
	// Subscribe to the streaming queries
	err := Subscribe(data, pks, msg.RID)
	if err != nil {
		Route(&Message{
			RID:     msg.RID,
			Target:  msg.ID,
			ID:      "subscribe",
			Payload: err.Error(),
			Type:    "response",
		})
	}
	Route(&Message{
		RID:     msg.RID,
		Target:  msg.ID,
		ID:      "subscribe",
		Payload: "success",
		Type:    "response",
	})
}

func Subscribe(r request, pks []string, connID string) error {
	streamsMu.Lock()
	defer streamsMu.Unlock()

	for _, pk := range pks {
		if streams[pk] == nil {
			streams[pk] = make(map[string]map[string]*QueryConnMap)
		}
		if streams[pk][r.Protopass] == nil {
			streams[pk][r.Protopass] = make(map[string]*QueryConnMap)
		}
		//loop queries and get object matching with context key and values else create new
		var connMap *QueryConnMap = nil
		for _, v := range streams[pk][r.Protopass] {
			if v.Ctxkey == r.Ctxkey && reflect.DeepEqual(v.Ctxvalues, r.Ctxvalues) {
				connMap = v
				break
			}
		}

		if connMap == nil {
			streams[pk][r.Protopass][r.Query] = &QueryConnMap{
				Ctxkey:    r.Ctxkey,
				Ctxvalues: r.Ctxvalues,
				Conns:     map[string]bool{connID: true},
			}
		} else {
			connMap.Ctxkey = r.Ctxkey
			connMap.Ctxvalues = r.Ctxvalues
			connMap.Conns[connID] = true
		}

		eval, err := SetupEvaluator(r.Protopass, r.Query, r.Ctxkey, r.Ctxvalues)
		if err != nil {
			return err
		}
		streams[pk][r.Protopass][r.Query].Eval = eval
	}
	return nil
}

func SetupEvaluator(protopass, query string, ctxkey string, ctxvalues []string) (*evaluator.Evaluator, error) {
	lexer := parser.NewLexer(query)
	plan := parser.NewPlan(lexer, protopass)
	if err := plan.Parse(); err != nil {
		return nil, err
	}
	ev := evaluator.NewEvaluator(plan, ctxkey, ctxvalues)
	return ev, nil
}

func Stream(data any, conns map[string]bool) {

	payload, _ := json.Marshal(data)
	// stream here
	msg := &Message{
		// RID: ,
		ID:      "subscribe",
		Target:  "server",
		Type:    "response",
		Payload: string(payload),
	}

	for conn, _ := range conns {
		msg.RID = conn
		Route(msg)
	}
}
