package model

type Signals struct {
	Openness              float64  `json:"openness"`
	Conscientiousness     float64  `json:"conscientiousness"`
	Extraversion          float64  `json:"extraversion"`
	Agreeableness         float64  `json:"agreeableness"`
	Neuroticism           float64  `json:"neuroticism"`
	OpennessConf          float64  `json:"openness_conf"`
	ConscientiousnessConf float64  `json:"conscientiousness_conf"`
	ExtraversionConf      float64  `json:"extraversion_conf"`
	AgreeablenessConf     float64  `json:"agreeableness_conf"`
	NeuroticismConf       float64  `json:"neuroticism_conf"`
	Interests             []string `json:"interests"`
	InterestConf          float64   `json:"interest_conf"`
	Evidence              []string  `json:"evidence"`
	Engagement            float64   `json:"engagement"`
	EngagementMedian      float64   `json:"engagement_median"`
	PostFrequency         float64   `json:"post_frequency"`
	Sentiment             float64   `json:"sentiment"` // -1..1
	TokenCount            int       `json:"token_count"`
	UniqueTokenCount      int       `json:"unique_token_count"`
	LexicalDiversity      float64   `json:"lexical_diversity"`
	ShannonEntropy        float64   `json:"shannon_entropy"`
	ShannonNormalized     float64   `json:"shannon_normalized"`
	ZipfSlope             float64   `json:"zipf_slope"`
	ZipfFit               float64   `json:"zipf_fit"`
	BayesConfidence       float64   `json:"bayes_confidence"`
}
