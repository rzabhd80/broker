package dbms

import "broker/internals/models"

type Dbms interface {
	Close() error
	StoreMessage(message models.Message, subject string) (int, error)
	FetchMessage(messageId int, subject string) (*models.Message, error)
}
