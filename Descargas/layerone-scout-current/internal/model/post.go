package model

import "time"

type Post struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Likes     int       `json:"likes"`
	Reposts   int       `json:"reposts"`
	Replies   int       `json:"replies"`
	CreatedAt time.Time `json:"created_at"`
	Hashtags  []string  `json:"hashtags,omitempty"`
	Mentions  []string  `json:"mentions,omitempty"`
	Links     []string  `json:"links,omitempty"`
	Media     bool      `json:"media"`
	Language  string    `json:"language,omitempty"`
}
