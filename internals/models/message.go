package models

import "time"

type Message struct {
	Id         int           `json:"id"`
	Body       string        `json:"body"`
	Subject    string        `json:"subject"`
	Expiration time.Duration `json:"expiration"`
}

type MessageStorage struct {
	Body       string    `json:"body"`
	Expiration time.Time `json:"expiration"`
}
