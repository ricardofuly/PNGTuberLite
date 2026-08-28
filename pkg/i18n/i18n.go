package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed locales/*.json
var embeddedLocales embed.FS

// LanguageMeta describes an available UI translation.
type LanguageMeta struct {
	Code string `json:"code"` // e.g. "pt-BR", "en-US"
	Name string `json:"name"` // e.g. "Português (Brasil)", "English (US)"
	Flag string `json:"flag"` // e.g. "🇧🇷", "🇺🇸"
}

// LocaleBundle contains language metadata and key-value string dictionary.
type LocaleBundle struct {
	Meta    LanguageMeta      `json:"meta"`
	Strings map[string]string `json:"strings"`
}

var (
	mu               sync.RWMutex
	currentLang      = "pt-BR"
	fallbackLang     = "pt-BR"
	languages        = make(map[string]*LocaleBundle)
	orderedLanguages = make([]LanguageMeta, 0)
)

func init() {
	LoadLocales()
}

// LoadLocales loads all embedded translation bundles and checks for any external custom JSON locale files.
func LoadLocales() {
	mu.Lock()
	defer mu.Unlock()

	languages = make(map[string]*LocaleBundle)
	orderedLanguages = make([]LanguageMeta, 0)

	// 1. Load embedded locales
	entries, err := embeddedLocales.ReadDir("locales")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				data, err := embeddedLocales.ReadFile("locales/" + entry.Name())
				if err == nil {
					var bundle LocaleBundle
					if err := json.Unmarshal(data, &bundle); err == nil && bundle.Meta.Code != "" {
						languages[bundle.Meta.Code] = &bundle
					}
				}
			}
		}
	}

	// 2. Load optional external custom locales from local ./locales directory
	if extEntries, err := os.ReadDir("locales"); err == nil {
		for _, entry := range extEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				data, err := os.ReadFile(filepath.Join("locales", entry.Name()))
				if err == nil {
					var bundle LocaleBundle
					if err := json.Unmarshal(data, &bundle); err == nil && bundle.Meta.Code != "" {
						languages[bundle.Meta.Code] = &bundle
					}
				}
			}
		}
	}

	// 3. Build sorted list of available languages
	for _, bundle := range languages {
		orderedLanguages = append(orderedLanguages, bundle.Meta)
	}

	sort.Slice(orderedLanguages, func(i, j int) bool {
		if orderedLanguages[i].Code == "pt-BR" {
			return true
		}
		if orderedLanguages[j].Code == "pt-BR" {
			return false
		}
		return orderedLanguages[i].Name < orderedLanguages[j].Name
	})

	// Detect system language if not set
	if currentLang == "" {
		currentLang = DetectSystemLanguage()
	}
}

// SetLanguage changes the active language.
func SetLanguage(code string) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := languages[code]; exists {
		currentLang = code
		return
	}

	// Try partial prefix match (e.g. "en" -> "en-US", "pt" -> "pt-BR")
	lower := strings.ToLower(code)
	for k := range languages {
		if strings.HasPrefix(strings.ToLower(k), lower) || strings.HasPrefix(lower, strings.ToLower(k)) {
			currentLang = k
			return
		}
	}

	currentLang = fallbackLang
}

// GetLanguage returns the current language code.
func GetLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// GetLanguageMeta returns metadata for the active language.
func GetLanguageMeta(code string) LanguageMeta {
	mu.RLock()
	defer mu.RUnlock()
	if b, exists := languages[code]; exists {
		return b.Meta
	}
	return LanguageMeta{Code: code, Name: code, Flag: "🌐"}
}

// GetAvailableLanguages returns the list of all registered translations.
func GetAvailableLanguages() []LanguageMeta {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]LanguageMeta, len(orderedLanguages))
	copy(result, orderedLanguages)
	return result
}

// T returns the translated string for the given key, optionally formatted with fmt.Sprintf arguments.
func T(key string, args ...interface{}) string {
	mu.RLock()
	bundle, exists := languages[currentLang]
	if !exists {
		bundle = languages[fallbackLang]
	}

	var raw string
	if bundle != nil {
		if val, found := bundle.Strings[key]; found {
			raw = val
		}
	}

	// Fallback to default bundle if key is missing in active translation
	if raw == "" && currentLang != fallbackLang {
		if fbBundle, fbExists := languages[fallbackLang]; fbExists {
			raw = fbBundle.Strings[key]
		}
	}
	mu.RUnlock()

	if raw == "" {
		raw = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(raw, args...)
	}
	return raw
}

// DetectSystemLanguage inspects environment variables to infer the system's locale.
func DetectSystemLanguage() string {
	langEnv := os.Getenv("LANG")
	if langEnv == "" {
		langEnv = os.Getenv("LC_ALL")
	}
	if langEnv == "" {
		langEnv = os.Getenv("LC_MESSAGES")
	}

	langLower := strings.ToLower(langEnv)
	if strings.Contains(langLower, "pt") {
		return "pt-BR"
	}
	if strings.Contains(langLower, "en") {
		return "en-US"
	}
	return "pt-BR"
}
