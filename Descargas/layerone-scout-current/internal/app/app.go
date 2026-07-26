package app

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"layerone-scout/internal/analyzer"
	"layerone-scout/internal/config"
	"layerone-scout/internal/fetcher"
	"layerone-scout/internal/model"
	"layerone-scout/internal/storage"
	"layerone-scout/pkg/utils"
)

var ErrNotFound = errors.New("persona no encontrada")

type App struct {
	store storage.Store
	cfg   config.Config
}

func New(store storage.Store, cfg config.Config) *App {
	return &App{store: store, cfg: cfg}
}

func (a *App) AddPerson(name, username, platform, bio string, followers, following int) (model.Person, error) {
	p := model.NewPerson(name, username, platform, bio, followers, following)
	_, err := a.store.Mutate(func(db *storage.Database) error {
		db.Persons = append(db.Persons, p)
		return nil
	})
	return p, err
}

func (a *App) ListPersons() ([]model.Person, error) {
	db, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return db.Persons, nil
}

func (a *App) GetPerson(key string) (model.Person, error) {
	db, err := a.store.Load()
	if err != nil {
		return model.Person{}, err
	}
	idx := findPersonIndex(db.Persons, key)
	if idx < 0 {
		return model.Person{}, ErrNotFound
	}
	return db.Persons[idx], nil
}

func (a *App) FetchAndAddProfile(ctx context.Context, platform, username string) (model.Person, error) {
	p, err := fetcher.FetchByPlatform(ctx, platform, username)
	if err != nil {
		return model.Person{}, err
	}
	db, _ := a.store.Load()
	idx := findPersonBySource(db.Persons, p.SourceID, p.Username, p.Platform)
	if idx >= 0 {
		updated := db.Persons[idx]
		existingPosts := make(map[string]bool)
		for _, post := range updated.Posts {
			existingPosts[post.ID] = true
		}
		for _, post := range p.Posts {
			if !existingPosts[post.ID] {
				updated.Posts = append(updated.Posts, post)
			}
		}
		updated.Followers = p.Followers
		updated.Following = p.Following
		updated.Bio = p.Bio
		updated.Name = p.Name
		updated.Username = p.Username
		updated.SourceURL = p.SourceURL
		updated.SourceID = p.SourceID
		updated.RawPostsCount = len(updated.Posts)
		updated.LastFetched = time.Now()
		updated = analyzer.AnalyzePerson(updated)
		_, err := a.store.Mutate(func(db *storage.Database) error {
			db.Persons[idx] = updated
			return nil
		})
		return updated, err
	}
	p = analyzer.AnalyzePerson(p)
	_, err = a.store.Mutate(func(db *storage.Database) error {
		db.Persons = append(db.Persons, p)
		return nil
	})
	return p, err
}

func (a *App) AnalyzePerson(key string) (model.Person, error) {
	var updated model.Person
	_, err := a.store.Mutate(func(db *storage.Database) error {
		idx := findPersonIndex(db.Persons, key)
		if idx < 0 {
			return ErrNotFound
		}
		p := db.Persons[idx]
		p = analyzer.AnalyzePerson(p)
		db.Persons[idx] = p
		updated = p
		return nil
	})
	return updated, err
}

func (a *App) ReportPerson(key string) (string, error) {
	p, err := a.GetPerson(key)
	if err != nil {
		return "", err
	}
	return generateReport(p), nil
}

func (a *App) FindPersonByName(name string) ([]model.Person, error) {
	db, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	name = strings.ToLower(strings.TrimSpace(name))
	var results []model.Person
	for _, p := range db.Persons {
		if strings.Contains(strings.ToLower(p.Name), name) || strings.Contains(strings.ToLower(p.Username), name) {
			results = append(results, p)
		}
	}
	return results, nil
}

func (a *App) ImportCSV(r io.Reader) (int, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(row []string, key string) string {
		i, ok := col[key]
		if !ok || i >= len(row) {
			return ""
		}
		return row[i]
	}
	var count int
	_, err = a.store.Mutate(func(db *storage.Database) error {
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			username := get(row, "username")
			if username == "" {
				continue
			}
			platform := get(row, "platform")
			if platform == "" {
				platform = "unknown"
			}
			postText := get(row, "post_text")
			likes := utils.ParseAbbreviated(get(row, "likes"))
			reposts := utils.ParseAbbreviated(get(row, "reposts"))
			createdAt, _ := utils.ParseTimeRFC3339(get(row, "created_at"))

			idx := findPersonByUsername(db.Persons, username, platform)
			var p model.Person
			if idx >= 0 {
				p = db.Persons[idx]
			} else {
				p = model.NewPerson("", username, platform, "", 0, 0)
				db.Persons = append(db.Persons, p)
				idx = len(db.Persons) - 1
			}
			post := model.Post{
				ID:        fmt.Sprintf("%s-%d", username, len(p.Posts)+1),
				Text:      postText,
				Likes:     likes,
				Reposts:   reposts,
				CreatedAt: createdAt,
			}
			post.Hashtags, post.Mentions = extractMetadata(post.Text)
			p.Posts = append(p.Posts, post)
			p.UpdatedAt = time.Now()
			p.RawPostsCount = len(p.Posts)
			db.Persons[idx] = p
			count++
		}
		return nil
	})
	return count, err
}

func findPersonIndex(persons []model.Person, key string) int {
	key = strings.ToLower(strings.TrimSpace(key))
	for i, p := range persons {
		if strings.ToLower(p.ID) == key || strings.ToLower(p.Name) == key || strings.ToLower(p.Username) == key {
			return i
		}
	}
	return -1
}

func findPersonByUsername(persons []model.Person, username, platform string) int {
	username = strings.ToLower(username)
	platform = strings.ToLower(platform)
	for i, p := range persons {
		if strings.ToLower(p.Username) == username && strings.ToLower(p.Platform) == platform {
			return i
		}
	}
	return -1
}

func findPersonBySource(persons []model.Person, sourceID, username, platform string) int {
	sourceID = strings.ToLower(sourceID)
	username = strings.ToLower(username)
	platform = strings.ToLower(platform)
	for i, p := range persons {
		if strings.ToLower(p.SourceID) == sourceID || (strings.ToLower(p.Username) == username && strings.ToLower(p.Platform) == platform) {
			return i
		}
	}
	return -1
}

func extractMetadata(text string) (hashtags, mentions []string) {
	tokens := strings.Fields(text)
	for _, t := range tokens {
		if strings.HasPrefix(t, "#") && len(t) > 1 {
			hashtags = append(hashtags, strings.ToLower(t[1:]))
		} else if strings.HasPrefix(t, "@") && len(t) > 1 {
			mentions = append(mentions, strings.ToLower(t[1:]))
		}
	}
	return
}

func generateReport(p model.Person) string {
	var b strings.Builder
	b.WriteString("# Perfil Psicológico de " + p.Name + "

")
	b.WriteString("**Usuario:** " + p.Username + " (" + p.Platform + ")
")
	b.WriteString("**Bio:** " + p.Bio + "
")
	b.WriteString("**Seguidores:** " + strconv.Itoa(p.Followers) + " | **Siguiendo:** " + strconv.Itoa(p.Following) + "
")
	b.WriteString("**Última actualización:** " + p.UpdatedAt.Format("2006-01-02 15:04") + "

")

	b.WriteString("## Rasgos Big Five

")
	b.WriteString("| Rasgo | Puntuación | Confianza |
")
	b.WriteString("|-------|------------|-----------|
")
	fmt.Fprintf(&b, "| Apertura | %.2f | %.2f |
", p.Signals.Openness, p.Signals.OpennessConf)
	fmt.Fprintf(&b, "| Responsabilidad | %.2f | %.2f |
", p.Signals.Conscientiousness, p.Signals.ConscientiousnessConf)
	fmt.Fprintf(&b, "| Extraversión | %.2f | %.2f |
", p.Signals.Extraversion, p.Signals.ExtraversionConf)
	fmt.Fprintf(&b, "| Amabilidad | %.2f | %.2f |
", p.Signals.Agreeableness, p.Signals.AgreeablenessConf)
	fmt.Fprintf(&b, "| Neuroticismo | %.2f | %.2f |
", p.Signals.Neuroticism, p.Signals.NeuroticismConf)

	b.WriteString("
## Intereses

")
	if len(p.Signals.Interests) == 0 {
		b.WriteString("No se detectaron intereses claros.
")
	} else {
		for _, i := range p.Signals.Interests {
			b.WriteString("- " + i + "
")
		}
	}
	fmt.Fprintf(&b, "Confianza en intereses: %.2f

", p.Signals.InterestConf)

	b.WriteString("## Métricas de actividad

")
	fmt.Fprintf(&b, "- Engagement medio: %.2f (likes+reposts por post)
", p.Signals.Engagement)
	fmt.Fprintf(&b, "- Engagement mediana: %.2f
", p.Signals.EngagementMedian)
	fmt.Fprintf(&b, "- Frecuencia de posts: %.2f posts/día
", p.Signals.PostFrequency)
	fmt.Fprintf(&b, "- Total de posts: %d
", len(p.Posts))

	b.WriteString("
## Perfil psicológico

")
	b.WriteString("**Arquetipo principal:** " + string(p.Profile) + "
")
	b.WriteString("**Descripción:** " + p.Profile.Description() + "
")
	b.WriteString("**Confianza del perfil:** " + fmt.Sprintf("%.2f", p.Confidence) + "
")
	b.WriteString("**Sugerencia de comunicación:** " + p.Profile.CommunicationStyle() + "
")

	if len(p.Signals.Evidence) > 0 {
		b.WriteString("
## Evidencia textual

")
		for _, e := range p.Signals.Evidence {
			b.WriteString("- " + e + "
")
		}
	}
	return b.String()
}
