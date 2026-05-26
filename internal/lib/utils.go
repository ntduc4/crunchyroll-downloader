package lib

import "sort"

var languageNames = map[string]string{
	"en-US":  "English",
	"en-IN":  "English (India)",
	"id-ID":  "Bahasa Indonesia",
	"ms-MY":  "Bahasa Melayu",
	"ca-ES":  "Català",
	"de-DE":  "Deutsch",
	"es-419": "Español (América Latina)",
	"es-ES":  "Español (España)",
	"fr-FR":  "Français",
	"it-IT":  "Italiano",
	"pl-PL":  "Polski",
	"pt-BR":  "Português (Brasil)",
	"pt-PT":  "Português (Portugal)",
	"vi-VN":  "Tiếng Việt",
	"tr-TR":  "Türkçe",
	"ru-RU":  "Русский",
	"ar-SA":  "العربية",
	"hi-IN":  "हिंदी",
	"ta-IN":  "தமிழ்",
	"te-IN":  "తెలుగు",
	"zh-CN":  "中文 (普通话)",
	"zh-HK":  "中文 (粵語)",
	"zh-TW":  "中文 (國語)",
	"ko-KR":  "한국어",
	"th-TH":  "ไทย",
}

func languageLabel(locale string) string {
	if name, ok := languageNames[locale]; ok {
		return name
	}

	return locale
}

func sortedLanguageKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
