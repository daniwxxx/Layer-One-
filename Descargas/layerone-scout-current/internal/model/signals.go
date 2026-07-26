package model

type Signals struct {
	Openness              float64   `json:"openness"`
	Conscientiousness     float64   `json:"conscientiousness"`
	Extraversion          float64   `json:"extraversion"`
	Agreeableness         float64   `json:"agreeableness"`
	Neuroticism           float64   `json:"neuroticism"`
	OpennessConf          float64   `json:"openness_conf"`
	ConscientiousnessConf float64   `json:"conscientiousness_conf"`
	ExtraversionConf      float64   `json:"extraversion_conf"`
	AgreeablenessConf     float64   `json:"agreeableness_conf"`
	NeuroticismConf       float64   `json:"neuroticism_conf"`
	Interests             []string  `json:"interests"`
	InterestConf          float64   `json:"interest_conf"`
	Evidence              []string  `json:"evidence"`
	Engagement            float64   `json:"engagement"`
	EngagementMedian      float64   `json:"engagement_median"`
	PostFrequency         float64   `json:"post_frequency"`
	Sentiment             float64   `json:"sentiment"` // -1..1
}
