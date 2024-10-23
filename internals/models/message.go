package models

import "time"

type Message struct {
	Id         int           `json:"id"`
	Body       byte          `json:"body"`
	Subject    string        `json:"subject"`
	Expiration time.Duration `json:"expiration"`
}
