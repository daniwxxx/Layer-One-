package cli

import "strings"

func bioOrFallback(bio string) string {
	if strings.TrimSpace(bio) == "" {
		return "(sin bio textual extraída)"
	}
	return bio
}
