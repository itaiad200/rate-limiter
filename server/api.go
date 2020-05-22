package server

import "time"

// Update is the message sent between servers to sync the db
type Update struct {
	UsersAccess map[string]Access
}

// Access represents a single user access
type Access struct{
	LastRequest time.Time
	Count int
}
