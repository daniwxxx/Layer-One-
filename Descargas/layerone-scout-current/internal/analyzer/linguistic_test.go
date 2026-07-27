package analyzer

import (
	"testing"
	"time"

	"layerone-scout/internal/model"
)

func TestAnalyzeCorpusMetrics(t *testing.T) {
	metrics, _ := AnalyzeCorpus([]string{
		"Creativo creativo #go @openai",
		"Ordenado y social, con ideas innovadoras",
	})
	if metrics.TokenCount == 0 {
		t.Fatal("expected token count > 0")
	}
	if metrics.UniqueTokenCount == 0 {
		t.Fatal("expected unique token count > 0")
	}
	if metrics.LexicalDiversity <= 0 || metrics.LexicalDiversity > 1 {
		t.Fatalf("lexical diversity out of range: %v", metrics.LexicalDiversity)
	}
	if metrics.ShannonNormalized < 0 || metrics.ShannonNormalized > 1 {
		t.Fatalf("shannon normalized out of range: %v", metrics.ShannonNormalized)
	}
}

func TestComputeBigFive_RecognizesLexicalSignals(t *testing.T) {
	scores, conf := ComputeBigFive([]string{
		"creativo innovador curioso",
		"social empático sociable",
	})
	if scores["openness"] <= 0.5 {
		t.Fatalf("expected openness > 0.5, got %v", scores["openness"])
	}
	if scores["extraversion"] <= 0.5 {
		t.Fatalf("expected extraversion > 0.5, got %v", scores["extraversion"])
	}
	if conf["openness"] == 0 && conf["extraversion"] == 0 {
		t.Fatal("expected non-zero confidence for lexical evidence")
	}
}

func TestAnalyzeBehavior_RecognizesCadenceAndStyle(t *testing.T) {
	posts := []model.Post{
		{
			Text:      "Explorando ideas nuevas! #go @openai",
			Likes:     12,
			Reposts:   4,
			Replies:   1,
			Hashtags:  []string{"go"},
			Mentions:  []string{"openai"},
			Links:     []string{"https://example.com"},
			CreatedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			Text:      "¿Qué opinan de esto? Me interesa mucho.",
			Likes:     8,
			Reposts:   2,
			Replies:   3,
			Hashtags:  []string{"opinion"},
			CreatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			Text:      "Seguimos construyendo con foco y orden.",
			Likes:     15,
			Reposts:   3,
			Replies:   0,
			CreatedAt: time.Date(2024, 1, 4, 8, 0, 0, 0, time.UTC),
		},
	}
	texts := []string{posts[0].Text, posts[1].Text, posts[2].Text}
	m := AnalyzeBehavior(posts, texts)
	if m.Burstiness == 0 && m.Regularity == 0 {
		t.Fatal("expected behavioral cadence metrics")
	}
	if m.QuestionRate == 0 || m.ExclamationRate == 0 {
		t.Fatal("expected punctuation-based behavior signals")
	}
	if m.ActiveSpanDays <= 0 {
		t.Fatal("expected positive active span")
	}
}

func TestClassifyProfile_UsesProbabilisticSignals(t *testing.T) {
	p, alts, conf := ClassifyProfile(model.Signals{
		Openness:          0.82,
		Conscientiousness:  0.20,
		Extraversion:      0.76,
		Agreeableness:     0.70,
		Neuroticism:       0.18,
		BayesConfidence:   0.64,
		LexicalDiversity:  0.42,
		ShannonNormalized: 0.61,
		ZipfFit:           0.73,
		Engagement:        3.4,
		Burstiness:        0.44,
		Regularity:        0.56,
		QuestionRate:      0.22,
		ExclamationRate:   0.18,
		MentionRate:       0.33,
		LinkRate:          0.17,
		AvgPostLength:     24,
		PostsPerActiveDay: 1.8,
	})
	if p == model.ProfileIndeterminado {
		t.Fatal("expected a concrete profile, got indeterminate")
	}
	if len(alts) == 0 {
		t.Fatal("expected alternatives to be populated")
	}
	if conf <= 0 {
		t.Fatalf("expected confidence > 0, got %v", conf)
	}
}
