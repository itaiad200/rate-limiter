package server

import "time"

// Update is the message sent between servers to sync the db
type Update struct {
	UsersAccess []Access
}

// Access represents a single user access
type Access struct{
	ClientID string
	LastRequest time.Time
	Count int
}
