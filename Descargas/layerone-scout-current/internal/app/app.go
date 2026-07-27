package app

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
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
	persons := append([]model.Person(nil), db.Persons...)
	sort.SliceStable(persons, func(i, j int) bool {
		if persons[i].UpdatedAt.Equal(persons[j].UpdatedAt) {
			return strings.ToLower(persons[i].Name) < strings.ToLower(persons[j].Name)
		}
		return persons[i].UpdatedAt.After(persons[j].UpdatedAt)
	})
	return persons, nil
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

func (a *App) DeletePerson(key string) (model.Person, error) {
	var deleted model.Person
	_, err := a.store.Mutate(func(db *storage.Database) error {
		idx := findPersonIndex(db.Persons, key)
		if idx < 0 {
			return ErrNotFound
		}
		deleted = db.Persons[idx]
		db.Persons = append(db.Persons[:idx], db.Persons[idx+1:]...)
		return nil
	})
	return deleted, err
}

func (a *App) FetchAndAddProfile(ctx context.Context, platform, username string) (model.Person, error) {
	p, err := fetcher.FetchByPlatform(ctx, platform, username)
	if err != nil {
		return model.Person{}, err
	}
	db, err := a.store.Load()
	if err != nil {
		return model.Person{}, err
	}
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
		updated.Platform = p.Platform
		updated.SourceURL = p.SourceURL
		updated.SourceID = p.SourceID
		updated.RawPostsCount = len(updated.Posts)
		updated.LastFetched = p.LastFetched
		updated = analyzer.AnalyzePerson(updated)
		updated.UpdatedAt = time.Now()
		_, err := a.store.Mutate(func(db *storage.Database) error {
			db.Persons[idx] = updated
			return nil
		})
		return updated, err
	}
	p = analyzer.AnalyzePerson(p)
	p.UpdatedAt = time.Now()
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
		p.UpdatedAt = time.Now()
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
	q := normalizeQuery(name)
	var results []model.Person
	for _, p := range db.Persons {
		nameNorm := normalizeQuery(p.Name)
		userNorm := normalizeQuery(p.Username)
		if q == "" {
			continue
		}
		if strings.Contains(nameNorm, q) || strings.Contains(userNorm, q) || strings.Contains(q, nameNorm) || strings.Contains(q, userNorm) {
			results = append(results, p)
			continue
		}
		if utils.Levenshtein(nameNorm, q) <= 2 || utils.Levenshtein(userNorm, q) <= 2 {
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
			username := strings.TrimSpace(get(row, "username"))
			if username == "" {
				continue
			}
			platform := strings.TrimSpace(get(row, "platform"))
			if platform == "" {
				platform = "unknown"
			}
			postText := utils.Sanitize(get(row, "post_text"))
			likes := utils.ParseAbbreviated(get(row, "likes"))
			reposts := utils.ParseAbbreviated(get(row, "reposts"))
			createdAt, err := utils.ParseTimeRFC3339(get(row, "created_at"))
			if err != nil || createdAt.IsZero() {
				createdAt = time.Now()
			}

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
	q := normalizeQuery(key)
	for i, p := range persons {
		if strings.EqualFold(strings.TrimSpace(p.ID), strings.TrimSpace(key)) || strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(key)) || strings.EqualFold(strings.TrimSpace(p.Username), strings.TrimSpace(key)) {
			return i
		}
		if q != "" {
			if normalizeQuery(p.ID) == q || normalizeQuery(p.Name) == q || normalizeQuery(p.Username) == q {
				return i
			}
		}
	}
	return -1
}

func findPersonByUsername(persons []model.Person, username, platform string) int {
	username = strings.ToLower(strings.TrimSpace(username))
	platform = strings.ToLower(strings.TrimSpace(platform))
	for i, p := range persons {
		if strings.ToLower(strings.TrimSpace(p.Username)) == username && strings.ToLower(strings.TrimSpace(p.Platform)) == platform {
			return i
		}
	}
	return -1
}

func findPersonBySource(persons []model.Person, sourceID, username, platform string) int {
	sourceID = strings.ToLower(strings.TrimSpace(sourceID))
	username = strings.ToLower(strings.TrimSpace(username))
	platform = strings.ToLower(strings.TrimSpace(platform))
	for i, p := range persons {
		if strings.ToLower(strings.TrimSpace(p.SourceID)) == sourceID || (strings.ToLower(strings.TrimSpace(p.Username)) == username && strings.ToLower(strings.TrimSpace(p.Platform)) == platform) {
			return i
		}
	}
	return -1
}

func normalizeQuery(s string) string {
	return strings.TrimSpace(strings.ToLower(utils.NormalizeText(s)))
}

func extractMetadata(text string) (hashtags, mentions []string) {
	tokens := strings.Fields(text)
	for _, t := range tokens {
		if strings.HasPrefix(t, "#") && len(t) > 1 {
			hashtags = append(hashtags, strings.ToLower(strings.Trim(t[1:], ".,;:!?()[]{}\"'")))
		} else if strings.HasPrefix(t, "@") && len(t) > 1 {
			mentions = append(mentions, strings.ToLower(strings.Trim(t[1:], ".,;:!?()[]{}\"'")))
		}
	}
	return
}

func generateReport(p model.Person) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Perfil Psicológico de %s\n\n", p.Name)
	fmt.Fprintf(&b, "**Usuario:** %s (%s)\n", p.Username, p.Platform)
	fmt.Fprintf(&b, "**Origen:** %s\n", firstNonEmpty(p.SourceURL, p.SourceID))
	fmt.Fprintf(&b, "**Bio:** %s\n", bioOrFallback(p.Bio))
	fmt.Fprintf(&b, "**Seguidores:** %d | **Siguiendo:** %d\n", p.Followers, p.Following)
	fmt.Fprintf(&b, "**Creado:** %s\n", p.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "**Última actualización:** %s\n\n", p.UpdatedAt.Format("2006-01-02 15:04"))

	fmt.Fprintf(&b, "## Rasgos Big Five\n\n")
	fmt.Fprintf(&b, "| Rasgo | Puntuación | Confianza |\n")
	fmt.Fprintf(&b, "|-------|------------|-----------|\n")
	fmt.Fprintf(&b, "| Apertura | %.2f | %.2f |\n", p.Signals.Openness, p.Signals.OpennessConf)
	fmt.Fprintf(&b, "| Responsabilidad | %.2f | %.2f |\n", p.Signals.Conscientiousness, p.Signals.ConscientiousnessConf)
	fmt.Fprintf(&b, "| Extraversión | %.2f | %.2f |\n", p.Signals.Extraversion, p.Signals.ExtraversionConf)
	fmt.Fprintf(&b, "| Amabilidad | %.2f | %.2f |\n", p.Signals.Agreeableness, p.Signals.AgreeablenessConf)
	fmt.Fprintf(&b, "| Neuroticismo | %.2f | %.2f |\n", p.Signals.Neuroticism, p.Signals.NeuroticismConf)

	fmt.Fprintf(&b, "\n## Métricas lingüísticas\n\n")
	fmt.Fprintf(&b, "- Tokens analizados: %d\n", p.Signals.TokenCount)
	fmt.Fprintf(&b, "- Vocabulario único: %d\n", p.Signals.UniqueTokenCount)
	fmt.Fprintf(&b, "- Diversidad léxica: %.2f\n", p.Signals.LexicalDiversity)
	fmt.Fprintf(&b, "- Entropía de Shannon: %.2f (normalizada %.2f)\n", p.Signals.ShannonEntropy, p.Signals.ShannonNormalized)
	fmt.Fprintf(&b, "- Zipf slope: %.2f | ajuste Zipf: %.2f\n", p.Signals.ZipfSlope, p.Signals.ZipfFit)
	fmt.Fprintf(&b, "- Confianza bayesiana: %.2f\n", p.Signals.BayesConfidence)

	fmt.Fprintf(&b, "\n## Intereses\n\n")
	if len(p.Signals.Interests) == 0 {
		fmt.Fprintf(&b, "No se detectaron intereses claros.\n")
	} else {
		for _, i := range p.Signals.Interests {
			fmt.Fprintf(&b, "- %s\n", i)
		}
	}
	fmt.Fprintf(&b, "Confianza en intereses: %.2f\n\n", p.Signals.InterestConf)

	fmt.Fprintf(&b, "## Métricas de comportamiento\n\n")
	fmt.Fprintf(&b, "- Burstiness: %.2f | Regularidad: %.2f\n", p.Signals.Burstiness, p.Signals.Regularity)
	fmt.Fprintf(&b, "- Tasa de preguntas: %.2f | Tasa de exclamaciones: %.2f\n", p.Signals.QuestionRate, p.Signals.ExclamationRate)
	fmt.Fprintf(&b, "- Hashtags/post: %.2f | Menciones/post: %.2f | Links/post: %.2f\n", p.Signals.HashtagRate, p.Signals.MentionRate, p.Signals.LinkRate)
	fmt.Fprintf(&b, "- Longitud media de posts: %.2f tokens\n", p.Signals.AvgPostLength)
	fmt.Fprintf(&b, "- Ventana activa: %.2f días | Posts/día activa: %.2f\n", p.Signals.ActiveSpanDays, p.Signals.PostsPerActiveDay)
	fmt.Fprintf(&b, "- Sentence count: %d\n", p.Signals.SentenceCount)
	fmt.Fprintf(&b, "- Media rate: %.2f | Repost rate: %.2f | Reply rate: %.2f\n", p.Signals.MediaRate, p.Signals.RepostRate, p.Signals.ReplyRate)
	fmt.Fprintf(&b, "- First-person rate: %.2f | Second-person rate: %.2f\n", p.Signals.FirstPersonRate, p.Signals.SecondPersonRate)
	fmt.Fprintf(&b, "- Call-to-action rate: %.2f\n", p.Signals.CallToActionRate)
	fmt.Fprintf(&b, "- Temporal entropy: %.2f | Repetition score: %.2f\n", p.Signals.TemporalEntropy, p.Signals.RepetitionScore)

	fmt.Fprintf(&b, "\n## Métricas de actividad\n\n")
	fmt.Fprintf(&b, "- Engagement medio: %.2f (likes+reposts por post)\n", p.Signals.Engagement)
	fmt.Fprintf(&b, "- Engagement mediana: %.2f\n", p.Signals.EngagementMedian)
	fmt.Fprintf(&b, "- Frecuencia de posts: %.2f posts/día\n", p.Signals.PostFrequency)
	fmt.Fprintf(&b, "- Total de posts: %d\n", len(p.Posts))

	fmt.Fprintf(&b, "\n## Perfil psicológico\n\n")
	fmt.Fprintf(&b, "**Arquetipo principal:** %s\n", p.Profile)
	fmt.Fprintf(&b, "**Descripción:** %s\n", p.Profile.Description())
	fmt.Fprintf(&b, "**Confianza del perfil:** %.2f\n", p.Confidence)
	fmt.Fprintf(&b, "**Sugerencia de comunicación:** %s\n", p.Profile.CommunicationStyle())

	if len(p.Signals.Evidence) > 0 {
		fmt.Fprintf(&b, "\n## Evidencia textual\n\n")
		for _, e := range p.Signals.Evidence {
			fmt.Fprintf(&b, "- %s\n", e)
		}
	}
	return b.String()
}

func bioOrFallback(bio string) string {
	if strings.TrimSpace(bio) == "" {
		return "(sin bio textual extraída)"
	}
	return bio
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
