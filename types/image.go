package types

import "github.com/google/uuid"

type Image struct {
	ImageID    uuid.UUID `json:"image_id"`
	ImagePath  string    `json:"image_path"`
	SeriesName string    `json:"series_name"`
	MovieName  string    `json:"movie_name"`
	MovieTitle string    `json:"movie_title"`
}
