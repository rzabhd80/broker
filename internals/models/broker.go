package models

import "context"

type Broker interface {
	Publish(ctx context.Context, subject string, message Message) (uint, error)
	Subscribe(ctx context.Context, subject string) (<-chan Message, error)
	Fetch(ctx context.Context, subject string, id int) (Message, error)
}
