package analyzer

import (
	"sort"
	"strings"
	"time"

	"layerone-scout/internal/model"
	"layerone-scout/pkg/utils"
)

func AnalyzePerson(p model.Person) model.Person {
	texts := []string{p.Bio}
	for _, post := range p.Posts {
		texts = append(texts, post.Text)
	}
	scores, conf := ComputeBigFive(texts)
	s := model.Signals{
		Openness:              scores["openness"],
		Conscientiousness:     scores["conscientiousness"],
		Extraversion:          scores["extraversion"],
		Agreeableness:         scores["agreeableness"],
		Neuroticism:           scores["neuroticism"],
		OpennessConf:          conf["openness"],
		ConscientiousnessConf: conf["conscientiousness"],
		ExtraversionConf:      conf["extraversion"],
		AgreeablenessConf:     conf["agreeableness"],
		NeuroticismConf:       conf["neuroticism"],
		Sentiment:             AnalyzeSentiment(texts),
	}
	interests, interestConf := ExtractInterests(texts)
	s.Interests = interests
	s.InterestConf = interestConf

	if len(p.Posts) > 0 {
		var eng []int
		total := 0
		for _, post := range p.Posts {
			val := post.Likes + post.Reposts
			eng = append(eng, val)
			total += val
		}
		s.Engagement = float64(total) / float64(len(p.Posts))
		sort.Ints(eng)
		if len(eng)%2 == 0 {
			s.EngagementMedian = float64(eng[len(eng)/2-1]+eng[len(eng)/2]) / 2
		} else {
			s.EngagementMedian = float64(eng[len(eng)/2])
		}
	}
	if len(p.Posts) > 1 {
		first := p.Posts[0].CreatedAt
		last := p.Posts[len(p.Posts)-1].CreatedAt
		days := last.Sub(first).Hours() / 24
		if days > 0 {
			s.PostFrequency = float64(len(p.Posts)) / days
		}
	}
	p.Signals = s
	primary, _, confProf := ClassifyProfile(s)
	p.Profile = primary
	p.Confidence = confProf
	// Evidencia
	p.Signals.Evidence = extractEvidence(texts)
	p.LastAnalyzed = time.Now()
	return p
}

func extractEvidence(texts []string) []string {
	var ev []string
	for _, text := range texts {
		if len(text) > 10 && (strings.Contains(text, "creativo") || strings.Contains(text, "ordenado") || strings.Contains(text, "social")) {
			ev = append(ev, utils.Truncate(text, 80))
			if len(ev) >= 3 {
				break
			}
		}
	}
	if len(ev) == 0 && len(texts) > 0 {
		ev = append(ev, utils.Truncate(texts[0], 80))
	}
	return ev
}
