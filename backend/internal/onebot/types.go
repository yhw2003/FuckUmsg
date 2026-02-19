package onebot

import "encoding/json"

type Event struct {
	PostType    string      `json:"post_type"`
	MessageType string      `json:"message_type"`
	Time        int64       `json:"time"`
	SelfID      int64       `json:"self_id"`
	MessageID   interface{} `json:"message_id"`
	RawMessage  string      `json:"raw_message"`
	Message     interface{} `json:"message"`
	UserID      int64       `json:"user_id"`
	GroupID     int64       `json:"group_id"`
	Sender      struct {
		UserID   int64  `json:"user_id"`
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
	} `json:"sender"`
}

type MessageSegment struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func DecodeSegments(raw interface{}) ([]MessageSegment, bool) {
	if raw == nil {
		return nil, false
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, false
	}
	var segments []MessageSegment
	if err := json.Unmarshal(bytes, &segments); err != nil {
		return nil, false
	}
	return segments, true
}
