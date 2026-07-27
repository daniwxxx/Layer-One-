package analyzer

import (
	"math"
	"sort"

	"layerone-scout/internal/model"
)

func ClassifyProfile(s model.Signals) (model.Profile, []model.Profile, float64) {
	scores := map[model.Profile]float64{
		model.ProfileExplorador:  profileScoreExplorador(s),
		model.ProfileOrganizador:  profileScoreOrganizador(s),
		model.ProfileSocial:       profileScoreSocial(s),
		model.ProfileAnalítico:    profileScoreAnalitico(s),
		model.ProfileEstable:      profileScoreEstable(s),
		model.ProfileEmocional:    profileScoreEmocional(s),
	}

	if s.BayesConfidence == 0 && s.TokenCount == 0 && len(s.Evidence) == 0 {
		return model.ProfileIndeterminado, []model.Profile{model.ProfileIndeterminado}, 0.15
	}

	type scoredProfile struct {
		profile model.Profile
		score   float64
		prob    float64
	}
	pairs := make([]scoredProfile, 0, len(scores))
	for profile, score := range scores {
		pairs = append(pairs, scoredProfile{profile: profile, score: score})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score == pairs[j].score {
			return pairs[i].profile < pairs[j].profile
		}
		return pairs[i].score > pairs[j].score
	})

	maxScore := pairs[0].score
	sum := 0.0
	for i := range pairs {
		pairs[i].prob = math.Exp((pairs[i].score - maxScore) * 5)
		sum += pairs[i].prob
	}
	for i := range pairs {
		pairs[i].prob /= sum
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].prob == pairs[j].prob {
			return pairs[i].profile < pairs[j].profile
		}
		return pairs[i].prob > pairs[j].prob
	})

	primary := pairs[0].profile
	confidence := pairs[0].prob
	second := 0.0
	if len(pairs) > 1 {
		second = pairs[1].prob
	}

	alternatives := make([]model.Profile, 0, 3)
	for i := 1; i < len(pairs) && i < 4; i++ {
		alternatives = append(alternatives, pairs[i].profile)
	}

	if confidence < 0.28 {
		return model.ProfileIndeterminado, alternativesWithDefault(alternatives, model.ProfileIndeterminado), confidence
	}
	if confidence < 0.42 || (confidence-second) < 0.08 {
		primary = model.ProfileMixto
	}
	if len(alternatives) == 0 {
		alternatives = []model.Profile{primary}
	}
	confidence = clamp01(0.7*confidence + 0.3*s.BayesConfidence)
	return primary, alternatives, confidence
}

func profileScoreExplorador(s model.Signals) float64 {
	return 0.42*s.Openness + 0.18*s.Extraversion + 0.12*(1-s.Neuroticism) + 0.12*s.LexicalDiversity + 0.08*s.ZipfFit + 0.08*s.BayesConfidence
}

func profileScoreOrganizador(s model.Signals) float64 {
	return 0.42*s.Conscientiousness + 0.18*(1-s.Neuroticism) + 0.15*(1-s.ShannonNormalized) + 0.12*s.ZipfFit + 0.13*s.BayesConfidence
}

func profileScoreSocial(s model.Signals) float64 {
	return 0.40*s.Extraversion + 0.24*s.Agreeableness + 0.12*sentimentPositive(s.Sentiment) + 0.12*engagementBoost(s) + 0.12*s.BayesConfidence
}

func profileScoreAnalitico(s model.Signals) float64 {
	return 0.34*s.Openness + 0.22*s.Conscientiousness + 0.18*s.LexicalDiversity + 0.14*s.ZipfFit + 0.12*s.BayesConfidence
}

func profileScoreEstable(s model.Signals) float64 {
	return 0.40*(1-s.Neuroticism) + 0.22*s.Agreeableness + 0.14*sentimentPositive(s.Sentiment) + 0.12*(1-s.ShannonNormalized) + 0.12*s.BayesConfidence
}

func profileScoreEmocional(s model.Signals) float64 {
	return 0.46*s.Neuroticism + 0.18*(1-sentimentPositive(s.Sentiment)) + 0.12*(1-s.LexicalDiversity) + 0.12*(1-s.ZipfFit) + 0.12*(1-s.BayesConfidence)
}

func alternativesWithDefault(alts []model.Profile, def model.Profile) []model.Profile {
	if len(alts) == 0 {
		return []model.Profile{def}
	}
	return alts
}

func sentimentPositive(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func engagementBoost(s model.Signals) float64 {
	if s.Engagement <= 0 {
		return 0
	}
	return math.Min(1, math.Log1p(s.Engagement)/5)
}
