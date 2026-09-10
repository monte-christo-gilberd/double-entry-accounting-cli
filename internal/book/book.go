package book

import "time"

type Book struct {
	ID          int64
	Name        string
	Description *string
	CreatedAt   time.Time
}
