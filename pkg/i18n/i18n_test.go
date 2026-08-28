package i18n

import (
	"testing"
)

func TestI18nBundlesAndSwitching(t *testing.T) {
	LoadLocales()

	langs := GetAvailableLanguages()
	if len(langs) < 2 {
		t.Fatalf("expected at least 2 languages (pt-BR and en-US), got %d", len(langs))
	}

	// Test pt-BR
	SetLanguage("pt-BR")
	if GetLanguage() != "pt-BR" {
		t.Errorf("expected pt-BR, got %s", GetLanguage())
	}
	if T("tab_avatars") != "Avatares" {
		t.Errorf("expected 'Avatares', got %q", T("tab_avatars"))
	}
	if T("label_status_speaking") != "🗣 Falando (Boca Aberta)" {
		t.Errorf("expected '🗣 Falando (Boca Aberta)', got %q", T("label_status_speaking"))
	}

	// Test en-US
	SetLanguage("en-US")
	if GetLanguage() != "en-US" {
		t.Errorf("expected en-US, got %s", GetLanguage())
	}
	if T("tab_avatars") != "Avatars" {
		t.Errorf("expected 'Avatars', got %q", T("tab_avatars"))
	}
	if T("label_status_speaking") != "🗣 Speaking (Open Mouth)" {
		t.Errorf("expected '🗣 Speaking (Open Mouth)', got %q", T("label_status_speaking"))
	}

	// Test Fallback
	SetLanguage("pt-BR")
	if T("non_existent_key_123") != "non_existent_key_123" {
		t.Errorf("expected key itself for non-existent translation, got %q", T("non_existent_key_123"))
	}
}
