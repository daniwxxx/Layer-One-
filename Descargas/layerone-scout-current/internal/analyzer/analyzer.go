package analyzer

import (
	"sort"
	"strings"
	"time"

	"layerone-scout/internal/model"
	"layerone-scout/pkg/utils"
)

func AnalyzePerson(p model.Person) model.Person {
	texts := make([]string, 0, len(p.Posts)+1)
	texts = append(texts, p.Bio)
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
		engagements := make([]int, 0, len(p.Posts))
		total := 0
		for _, post := range p.Posts {
			val := post.Likes + post.Reposts
			engagements = append(engagements, val)
			total += val
		}
		s.Engagement = float64(total) / float64(len(p.Posts))
		sort.Ints(engagements)
		if len(engagements)%2 == 0 {
			s.EngagementMedian = float64(engagements[len(engagements)/2-1]+engagements[len(engagements)/2]) / 2
		} else {
			s.EngagementMedian = float64(engagements[len(engagements)/2])
		}
	}

	if len(p.Posts) > 1 {
		ordered := make([]model.Post, len(p.Posts))
		copy(ordered, p.Posts)
		sort.Slice(ordered, func(i, j int) bool {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		})
		first := ordered[0].CreatedAt
		last := ordered[len(ordered)-1].CreatedAt
		days := last.Sub(first).Hours() / 24
		if days > 0 {
			s.PostFrequency = float64(len(ordered)) / days
		}
	}

	p.Signals = s
	primary, _, confProf := ClassifyProfile(s)
	p.Profile = primary
	p.Confidence = confProf
	p.Signals.Evidence = extractEvidence(texts)
	p.LastAnalyzed = time.Now()
	return p
}

func extractEvidence(texts []string) []string {
	terms := []string{"creativo", "ordenado", "social", "innovador", "responsable", "empático", "analítico"}
	var ev []string
	for _, text := range texts {
		low := strings.ToLower(text)
		if len(low) <= 10 {
			continue
		}
		for _, term := range terms {
			if strings.Contains(low, term) {
				ev = append(ev, utils.Truncate(text, 80))
				break
			}
		}
		if len(ev) >= 3 {
			break
		}
	}
	if len(ev) == 0 && len(texts) > 0 {
		ev = append(ev, utils.Truncate(texts[0], 80))
	}
	return ev
}
