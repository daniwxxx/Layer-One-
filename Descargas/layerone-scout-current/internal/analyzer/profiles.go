package analyzer

import (
	"layerone-scout/internal/model"
)

func ClassifyProfile(s model.Signals) (model.Profile, []model.Profile, float64) {
	const high = 0.65
	const low = 0.35
	conditions := []struct {
		profile model.Profile
		cond    func() bool
	}{
		{model.ProfileExplorador, func() bool { return s.Openness > high && s.Extraversion > low && s.Neuroticism < 0.5 }},
		{model.ProfileOrganizador, func() bool { return s.Conscientiousness > high && s.Neuroticism < 0.4 }},
		{model.ProfileSocial, func() bool { return s.Extraversion > high && s.Agreeableness > high }},
		{model.ProfileAnalítico, func() bool { return s.Openness > high && s.Extraversion < low }},
		{model.ProfileEstable, func() bool { return s.Neuroticism < low && s.Agreeableness > 0.5 }},
		{model.ProfileEmocional, func() bool { return s.Neuroticism > high }},
	}
	var primary model.Profile
	var alternatives []model.Profile
	matched := 0
	for _, c := range conditions {
		if c.cond() {
			matched++
			if primary == "" {
				primary = c.profile
			} else {
				alternatives = append(alternatives, c.profile)
			}
		}
	}
	if matched == 0 {
		return model.ProfileIndeterminado, []model.Profile{model.ProfileIndeterminado}, 0.3
	}
	if matched > 1 {
		primary = model.ProfileMixto
	}
	conf := float64(matched) / float64(len(conditions))
	if conf > 1 {
		conf = 1
	}
	return primary, alternatives, conf
}
