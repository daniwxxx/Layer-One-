package analyzer

import (
	"testing"
	"time"

	"layerone-scout/internal/model"
)

func TestAnalyzeBehaviorPlus(t *testing.T) {
	posts := []model.Post{
		{Text: "Comenta qué opinas! #go @openai", Likes: 12, Reposts: 2, Replies: 1, Media: true, CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), Hashtags: []string{"go"}, Mentions: []string{"openai"}},
		{Text: "Seguimos construyendo con foco y orden.", Likes: 8, Reposts: 1, Replies: 0, CreatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)},
		{Text: "¿Qué piensas tú? Mira el link en bio.", Likes: 9, Reposts: 1, Replies: 3, CreatedAt: time.Date(2024, 1, 4, 8, 0, 0, 0, time.UTC), Links: []string{"https://example.com"}},
	}
	texts := []string{posts[0].Text, posts[1].Text, posts[2].Text}
	m := AnalyzeBehaviorPlus(posts, texts)
	if m.SentenceCount == 0 || m.MediaRate == 0 || m.CallToActionRate == 0 {
		t.Fatalf("expected richer behavioral features, got %#v", m)
	}
	if m.TemporalEntropy == 0 && len(posts) > 1 {
		t.Fatal("expected temporal entropy to be > 0")
	}
	if m.RepetitionScore < 0 {
		t.Fatal("repetition score should not be negative")
	}
}

func TestAnalyzeSentimentPlus(t *testing.T) {
	v := AnalyzeSentimentPlus([]string{"me encanta esto", "no me gusta nada"})
	if v == 0 {
		t.Fatal("expected sentiment signal")
	}
}

func TestClassifyProfilePlus(t *testing.T) {
	p, alts, conf := ClassifyProfilePlus(model.Signals{
		Openness:         0.84,
		Conscientiousness: 0.42,
		Extraversion:     0.79,
		Agreeableness:    0.71,
		Neuroticism:      0.18,
		BayesConfidence:  0.67,
		LexicalDiversity:  0.58,
		ShannonNormalized: 0.63,
		ZipfFit:          0.71,
		Burstiness:       0.44,
		Regularity:       0.56,
		QuestionRate:     0.19,
		ExclamationRate:  0.11,
		MentionRate:      0.22,
		MediaRate:        0.31,
		CallToActionRate: 0.15,
		TemporalEntropy:  0.48,
		RepetitionScore:  0.17,
	})
	if p == model.ProfileIndeterminado {
		t.Fatal("expected concrete profile")
	}
	if len(alts) == 0 {
		t.Fatal("expected alternatives")
	}
	if conf <= 0 {
		t.Fatalf("expected positive confidence, got %v", conf)
	}
}
