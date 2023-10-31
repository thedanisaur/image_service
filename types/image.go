package types

import "github.com/google/uuid"

type Image struct {
	ID        uuid.UUID `json:"image_id"`
	MovieName string    `json:"movie_name"`
}
