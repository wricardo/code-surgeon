package structs

import "time"

// Simple structure is a simple struct
// Represents a user record in the database
type SimpleStruct struct {
	// Id is the user's id
	ID             int        `db:"id"`
	Name           string     `db:"name"`
	FavoriteColors []string   `db:"favorite_colors"`
	DateUpdated    *time.Time `db:"date_updated"` // only if the record has been updated
}
