package main

import (
	"reflect"
	"testing"
)

func TestLanguageLabel(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "known locale en-US", locale: "en-US", want: "English"},
		{name: "known locale ja-JP", locale: "ja-JP", want: "ja-JP"},
		{name: "known locale pt-BR", locale: "pt-BR", want: "Português (Brasil)"},
		{name: "known locale de-DE", locale: "de-DE", want: "Deutsch"},
		{name: "unknown locale returns itself", locale: "xx-XX", want: "xx-XX"},
		{name: "empty locale", locale: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := languageLabel(tt.locale)
			if got != tt.want {
				t.Errorf("languageLabel(%q) = %q, want %q", tt.locale, got, tt.want)
			}
		})
	}
}

func TestSortedLanguageKeys(t *testing.T) {
	tests := []struct {
		name  string
		items map[string]int
		want  []string
	}{
		{name: "empty map", items: map[string]int{}, want: []string{}},
		{name: "nil map", items: nil, want: []string{}},
		{name: "single entry", items: map[string]int{"en-US": 1}, want: []string{"en-US"}},
		{name: "sorted ascending",
			items: map[string]int{
				"fr-FR": 1,
				"en-US": 2,
				"de-DE": 3,
			},
			want: []string{"de-DE", "en-US", "fr-FR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedLanguageKeys(tt.items)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortedLanguageKeys(%v) = %v, want %v", tt.items, got, tt.want)
			}
		})
	}
}

func TestSortedLanguageKeys_StringValues(t *testing.T) {
	items := map[string]string{
		"ja-JP": "x",
		"ar-SA": "x",
		"zh-CN": "x",
	}
	want := []string{"ar-SA", "ja-JP", "zh-CN"}
	got := sortedLanguageKeys(items)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortedLanguageKeys(%v) = %v, want %v", items, got, want)
	}
}
