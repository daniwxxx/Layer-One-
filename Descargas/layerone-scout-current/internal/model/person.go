package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Person struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Username      string    `json:"username"`
	Platform      string    `json:"platform"`
	Bio           string    `json:"bio"`
	Followers     int       `json:"followers"`
	Following     int       `json:"following"`
	SourceURL     string    `json:"source_url"`
	SourceID      string    `json:"source_id"`
	Posts         []Post    `json:"posts"`
	Signals       Signals   `json:"signals"`
	Profile       Profile   `json:"profile"`
	Confidence    float64   `json:"confidence"`
	RawPostsCount int       `json:"raw_posts_count"`
	LastFetched   time.Time `json:"last_fetched"`
	LastAnalyzed  time.Time `json:"last_analyzed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewPerson(name, username, platform, bio string, followers, following int) Person {
	return Person{
		ID:        newID(),
		Name:      name,
		Username:  username,
		Platform:  platform,
		Bio:       bio,
		Followers: followers,
		Following: following,
		Posts:     []Post{},
		Signals:   Signals{Interests: []string{}, Evidence: []string{}},
		Profile:   ProfileUnknown,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return hex.EncodeToString(b[:])
}
