package analyzer

import (
	"math"
	"sort"

	"layerone-scout/internal/model"
)

func ClassifyProfile(s model.Signals) (model.Profile, []model.Profile, float64) {
	scores := map[model.Profile]float64{
		model.ProfileExplorador:    profileScoreExplorador(s),
		model.ProfileOrganizador:   profileScoreOrganizador(s),
		model.ProfileSocial:        profileScoreSocial(s),
		model.ProfileAnalítico:     profileScoreAnalitico(s),
		model.ProfileEstable:       profileScoreEstable(s),
		model.ProfileEmocional:     profileScoreEmocional(s),
	}

	if s.BayesConfidence == 0 && s.TokenCount == 0 && len(s.Evidence) == 0 {
		return model.ProfileIndeterminado, []model.Profile{model.ProfileIndeterminado}, 0.15
	}

	type pair struct {
		profile model.Profile
		score   float64
	}
	pairs := make([]pair, 0, len(scores))
	for profile, score := range scores {
		pairs = append(pairs, pair{profile: profile, score: score})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].score == pairs[j].score {
			return pairs[i].profile < pairs[j].profile
		}
		return pairs[i].score > pairs[j].score
	})

	probs := softmaxScores(pairs)
	top := probs[0]
	second := 0.0
	if len(probs) > 1 {
		second = probs[1]
	}

	alternatives := make([]model.Profile, 0, 3)
	for i := 1; i < len(probs) && i < 4; i++ {
		alternatives = append(alternatives, probs[i].profile)
	}

	primary := probs[0].profile
	confidence := probs[0].prob
	if confidence < 0.34 || (confidence-second) < 0.08 {
		if confidence < 0.28 {
			return model.ProfileIndeterminado, alternativesWithDefault(alternatives, model.ProfileIndeterminado), confidence
		}
		primary = model.ProfileMixto
		confidence = math.Max(confidence, 0.34)
	}

	if len(alternatives) == 0 {
		alternatives = []model.Profile{primary}
	}
	if top.prob > 0 && confidence > 0 {
		confidence = clamp01(0.7*confidence + 0.3*s.BayesConfidence)
	}
	return primary, alternatives, confidence
}

func profileScoreExplorador(s model.Signals) float64 {
	return 0.42*s.Openness + 0.18*s.Extraversion + 0.12*(1-s.Neuroticism) + 0.12*s.LexicalDiversity + 0.08*s.ZipfFit + 0.08*s.BayesConfidence
}

func profileScoreOrganizador(s model.Signals) float64 {
	return 0.42*s.Conscientiousness + 0.18*(1-s.Neuroticism) + 0.15*(1-s.ShannonNormalized) + 0.12*s.ZipfFit + 0.13*s.BayesConfidence
}

func profileScoreSocial(s model.Signals) float64 {
	return 0.40*s.Extraversion + 0.24*s.Agreeableness + 0.12*sentimentPositive(s.Sentiment) + 0.12*s.EngagementBoost() + 0.12*s.BayesConfidence
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

type scoredProfile struct {
	profile model.Profile
	prob    float64
}

func softmaxScores(pairs []struct {
	profile model.Profile
	score   float64
}) []scoredProfile {
	if len(pairs) == 0 {
		return nil
	}
	maxScore := pairs[0].score
	for _, p := range pairs[1:] {
		if p.score > maxScore {
			maxScore = p.score
		}
	}
	exps := make([]float64, len(pairs))
	sum := 0.0
	for i, p := range pairs {
		e := math.Exp((p.score - maxScore) * 5)
		exps[i] = e
		sum += e
	}
	out := make([]scoredProfile, len(pairs))
	for i, p := range pairs {
		out[i] = scoredProfile{profile: p.profile, prob: exps[i] / sum}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].prob > out[j].prob })
	return out
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

func (s model.Signals) EngagementBoost() float64 {
	if s.Engagement <= 0 {
		return 0
	}
	return math.Min(1, math.Log1p(s.Engagement)/5)
}
