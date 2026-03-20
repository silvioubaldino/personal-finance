package usecase

import (
	"strings"
	"testing"
)

func TestDeriveConversationTitle(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", maxConversationTitleRunes+10)
	wantLong := strings.Repeat("a", maxConversationTitleRunes-1) + "…"

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "whitespace_only", in: " \n\t ", want: ""},
		{name: "simple", in: "  Como posso economizar?  ", want: "Como posso economizar?"},
		{name: "collapse_newlines", in: "linha1\n\nlinha2", want: "linha1 linha2"},
		{name: "truncate_ascii", in: long, want: wantLong},
		{name: "unicode_short", in: "Orçamento 日本", want: "Orçamento 日本"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deriveConversationTitle(tc.in)
			if got != tc.want {
				t.Fatalf("deriveConversationTitle(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
