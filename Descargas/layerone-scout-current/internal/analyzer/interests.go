package analyzer

import (
	"strings"

	"layerone-scout/pkg/utils"
)

var interestKeywords = map[string][]string{
	"tecnología":  {"tech", "software", "código", "programación", "inteligencia artificial", "ia", "machine learning", "blockchain", "startup", "app", "digital"},
	"deportes":    {"fútbol", "tenis", "baloncesto", "deporte", "correr", "gimnasio", "entrenamiento", "fitness", "yoga", "running"},
	"cultura":     {"arte", "música", "cine", "literatura", "pintura", "escultura", "teatro", "poesía", "libro", "lectura"},
	"negocios":    {"emprender", "startup", "inversión", "mercado", "ventas", "marketing", "finanzas", "economía", "negocio", "empresa"},
	"salud":       {"ejercicio", "nutrición", "meditación", "bienestar", "yoga", "salud mental", "terapia", "alimentación", "cuidado personal"},
	"viajes":      {"viajar", "turismo", "aventura", "destino", "mochilero", "playa", "montaña", "explorar", "vacaciones"},
	"ciencia":     {"investigación", "descubrimiento", "física", "biología", "química", "astronomía", "evolución", "genética"},
	"política":    {"gobierno", "votar", "partido", "democracia", "derechos", "justicia social", "activismo", "política", "elección"},
	"moda":        {"ropa", "estilo", "diseño", "tendencia", "zapatos", "accesorios", "look", "vestir"},
	"gastronomía": {"comida", "receta", "cocinar", "restaurante", "chef", "gourmet", "sabor", "plato"},
}

func ExtractInterests(texts []string) ([]string, float64) {
	interestCount := make(map[string]int)
	totalTokens := 0
	joined := normalizeForPhrase(strings.Join(texts, " "))
	for _, text := range texts {
		tokens := utils.Tokenize(text)
		totalTokens += len(tokens)
	}

	for topic, keywords := range interestKeywords {
		for _, kw := range keywords {
			kwNorm := normalizeForPhrase(kw)
			if kwNorm == "" {
				continue
			}
			occurrences := strings.Count(joined, kwNorm)
			if occurrences > 0 {
				interestCount[topic] += occurrences
			}
		}
	}

	var interests []string
	for topic, count := range interestCount {
		if count >= 2 {
			interests = append(interests, topic)
		}
	}
	for i := 0; i < len(interests)-1; i++ {
		for j := i + 1; j < len(interests); j++ {
			if interestCount[interests[i]] < interestCount[interests[j]] {
				interests[i], interests[j] = interests[j], interests[i]
			}
		}
	}
	if len(interests) > 5 {
		interests = interests[:5]
	}

	conf := 0.0
	if totalTokens > 0 {
		topicDensity := float64(len(interests)) / 5.0
		hitDensity := 0.0
		for _, count := range interestCount {
			hitDensity += float64(count)
		}
		hitDensity = clamp01(hitDensity / float64(totalTokens))
		conf = clamp01(0.55*topicDensity + 0.45*hitDensity)
	}
	return interests, conf
}

func normalizeForPhrase(s string) string {
	s = strings.ToLower(utils.NormalizeText(s))
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	return " " + s + " "
}
