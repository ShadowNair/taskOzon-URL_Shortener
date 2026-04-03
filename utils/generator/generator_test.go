package generator

import "testing"

func TestRandomGeneratorGenerate(t *testing.T) {
	g := &RandomGenerator{}
	got, err := g.Generate(10)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("Generate() length = %d", len(got))
	}
	for _, ch := range got {
		if !containsRune(Alphabet, ch) {
			t.Fatalf("invalid char generated: %q", ch)
		}
	}
}

func TestRandomGeneratorGenerateInvalidLength(t *testing.T) {
	g := &RandomGenerator{}
	if _, err := g.Generate(0); err == nil {
		t.Fatal("expected error for zero length")
	}
}

func containsRune(s string, r rune) bool {
	for _, ch := range s {
		if ch == r {
			return true
		}
	}
	return false
}