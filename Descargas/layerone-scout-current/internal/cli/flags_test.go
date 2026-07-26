package cli

import "testing"

func TestOptionalBool_DefaultsToFalseWhenNotPassed(t *testing.T) {
	fs := NewFlagSet("test")
	v := fs.OptionalBool("flag", "uso")
	if err := fs.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	if *v != false {
		t.Fatalf("sin pasar el flag, valor debería ser false, obtuve %v", *v)
	}
	if fs.WasSet("flag") {
		t.Fatal("WasSet debería ser false si no se pasó")
	}
}

func TestOptionalBool_TracksExplicitValue(t *testing.T) {
	fs := NewFlagSet("test")
	v := fs.OptionalBool("flag", "uso")
	if err := fs.Parse([]string{"--flag=false"}); err != nil {
		t.Fatal(err)
	}
	if *v != false {
		t.Fatalf("valor explícito false debería reflejarse, obtuve %v", *v)
	}
	if !fs.WasSet("flag") {
		t.Fatal("WasSet debería ser true si se pasó el flag")
	}
}
