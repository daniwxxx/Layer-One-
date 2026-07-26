package fetcher

import (
	"context"
	"fmt"
	"time"

	"layerone-scout/internal/model"
)

type MockFetcher struct{}

func NewMockFetcher() *MockFetcher { return &MockFetcher{} }

func (f *MockFetcher) Platform() string { return "mock" }

func (f *MockFetcher) Fetch(ctx context.Context, username string) (model.Person, error) {
	p := model.NewPerson("Usuario de prueba", username, "mock", "Bio de ejemplo", 1234, 567)
	p.SourceURL = "https://ejemplo.com/" + username
	p.SourceID = username
	for i := 0; i < 5; i++ {
		post := model.Post{
			ID:        fmt.Sprintf("%s-%d", username, i),
			Text:      "Este es un post de ejemplo para pruebas. #test",
			Likes:     10 + i*5,
			Reposts:   2 + i,
			CreatedAt: time.Now().Add(-time.Duration(i*2) * time.Hour),
			Hashtags:  []string{"test", "ejemplo"},
		}
		p.Posts = append(p.Posts, post)
	}
	p.RawPostsCount = len(p.Posts)
	p.LastFetched = time.Now()
	return p, nil
}
