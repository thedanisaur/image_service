package types

import "github.com/google/uuid"

type Image struct {
	ID         uuid.UUID `json:"image_id"`
	SeriesName string    `json:"series_name"`
	MovieName  string    `json:"movie_name"`
	MovieTitle string    `json:"movie_title"`
}
