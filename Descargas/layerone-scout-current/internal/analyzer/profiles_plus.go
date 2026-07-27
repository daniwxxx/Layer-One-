package analyzer

import (
	"math"
	"sort"

	"layerone-scout/internal/model"
)

func ClassifyProfilePlus(s model.Signals) (model.Profile, []model.Profile, float64) {
	scores := map[model.Profile]float64{
		model.ProfileExplorador: profileScoreExploradorPlus(s),
		model.ProfileOrganizador: profileScoreOrganizadorPlus(s),
		model.ProfileSocial:      profileScoreSocialPlus(s),
		model.ProfileAnalítico:   profileScoreAnaliticoPlus(s),
		model.ProfileEstable:     profileScoreEstablePlus(s),
		model.ProfileEmocional:   profileScoreEmocionalPlus(s),
	}

	if s.BayesConfidence == 0 && s.TokenCount == 0 && len(s.Evidence) == 0 && s.SentenceCount == 0 {
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

	if confidence < 0.24 {
		return model.ProfileIndeterminado, alternativesWithDefault(alternatives, model.ProfileIndeterminado), confidence
	}
	if confidence < 0.40 || (confidence-second) < 0.07 {
		primary = model.ProfileMixto
	}
	if len(alternatives) == 0 {
		alternatives = []model.Profile{primary}
	}
	confidence = clamp01(0.68*confidence + 0.32*behaviorEvidenceStrength(s))
	return primary, alternatives, confidence
}

func profileScoreExploradorPlus(s model.Signals) float64 {
	return clamp01(
		0.27*s.Openness +
		0.10*s.Extraversion +
		0.08*(1-s.Neuroticism) +
		0.10*s.LexicalDiversity +
		0.06*s.ZipfFit +
		0.06*s.BayesConfidence +
		0.09*s.Burstiness +
		0.06*s.MediaRate +
		0.06*s.TemporalEntropy +
		0.06*s.CallToActionRate +
		0.06*behaviorEvidenceStrength(s),
	)
}

func profileScoreOrganizadorPlus(s model.Signals) float64 {
	return clamp01(
		0.28*s.Conscientiousness +
		0.13*s.Regularity +
		0.10*(1-s.Burstiness) +
		0.10*(1-s.ShannonNormalized) +
		0.07*(1-s.ExclamationRate) +
		0.07*(1-s.QuestionRate) +
		0.07*(1-s.RepetitionScore) +
		0.06*s.BayesConfidence +
		0.06*postLengthBalance(s) +
		0.06*(1-s.Neuroticism),
	)
}

func profileScoreSocialPlus(s model.Signals) float64 {
	return clamp01(
		0.27*s.Extraversion +
		0.16*s.Agreeableness +
		0.10*s.MentionRate +
		0.08*s.FirstPersonRate +
		0.08*s.SecondPersonRate +
		0.08*s.CallToActionRate +
		0.08*engagementBoost(s) +
		0.06*sentimentPositive(s.Sentiment) +
		0.06*s.MediaRate +
		0.05*s.QuestionRate +
		0.04*s.ExclamationRate +
		0.04*behaviorEvidenceStrength(s),
	)
}

func profileScoreAnaliticoPlus(s model.Signals) float64 {
	return clamp01(
		0.20*s.Openness +
		0.15*s.Conscientiousness +
		0.12*s.LexicalDiversity +
		0.10*s.ZipfFit +
		0.10*s.LinkRate +
		0.08*postLengthBalance(s) +
		0.08*s.Regularity +
		0.07*(1-s.ExclamationRate) +
		0.07*(1-s.QuestionRate) +
		0.06*(1-s.MediaRate) +
		0.06*s.BayesConfidence +
		0.03*(1-s.RepetitionScore),
	)
}

func profileScoreEstablePlus(s model.Signals) float64 {
	return clamp01(
		0.26*(1-s.Neuroticism) +
		0.16*s.Agreeableness +
		0.12*s.Regularity +
		0.10*(1-s.Burstiness) +
		0.08*sentimentPositive(s.Sentiment) +
		0.08*(1-s.ExclamationRate) +
		0.06*(1-s.QuestionRate) +
		0.06*(1-s.TemporalEntropy) +
		0.06*(1-s.RepetitionScore) +
		0.06*s.BayesConfidence +
		0.06*(1-s.NeuroticismConf),
	)
}

func profileScoreEmocionalPlus(s model.Signals) float64 {
	return clamp01(
		0.30*s.Neuroticism +
		0.10*s.Burstiness +
		0.09*s.ExclamationRate +
		0.09*s.QuestionRate +
		0.08*(1-s.Regularity) +
		0.08*(1-sentimentPositive(s.Sentiment)) +
		0.08*(1-s.RepetitionScore) +
		0.06*s.FirstPersonRate +
		0.06*(1-s.MediaRate) +
		0.06*(1-s.BayesConfidence) +
		0.06*s.TemporalEntropy,
	)
}

func behaviorEvidenceStrength(s model.Signals) float64 {
	v := (s.Burstiness + s.Regularity + s.BayesConfidence + s.LexicalDiversity + s.ShannonNormalized) / 5
	return clamp01(v)
}
