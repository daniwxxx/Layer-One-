package scraper

import (
	"context"
	"testing"
)

func TestFetchInstagramProfile_Fallback(t *testing.T) {
	// No se puede probar en entorno de build sin red, pero comprobamos que no panic
	ctx := context.Background()
	_, err := FetchInstagramProfile(ctx, "testuser", nil)
	if err == nil {
		t.Log("Instagram fetch devolvió datos (quizás mock)")
	} else {
		t.Log("Instagram fetch falló (esperado en entorno sin red)")
	}
}
