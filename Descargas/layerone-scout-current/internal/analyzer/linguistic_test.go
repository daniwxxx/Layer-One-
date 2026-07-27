package analyzer

import (
	"testing"

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

func TestClassifyProfile_UsesProbabilisticSignals(t *testing.T) {
	p, alts, conf := ClassifyProfile(model.Signals{
		Openness:          0.82,
		Conscientiousness:  0.20,
		Extraversion:      0.76,
		Agreeableness:     0.70,
		Neuroticism:       0.18,
		BayesConfidence:   0.64,
		LexicalDiversity:   0.42,
		ShannonNormalized:  0.61,
		ZipfFit:           0.73,
		Engagement:        3.4,
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
