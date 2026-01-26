package models

// User represents a person or entity on the leaderboard.
type User struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}
