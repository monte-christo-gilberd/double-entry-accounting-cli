package account

import "time"

type Account struct {
	ID          int64
	BookID      int64
	Code        string
	Name        string
	AccountType string
	CreatedAt   time.Time
}
