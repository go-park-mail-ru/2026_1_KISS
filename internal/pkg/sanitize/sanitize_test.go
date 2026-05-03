package sanitize_test

import (
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/sanitize"
)

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"<img src=x onerror=alert(1)>", "&lt;img src=x onerror=alert(1)&gt;"},
		{"", ""},
		{"normal text 123", "normal text 123"},
		{"<b>bold</b> & 'it'", "&lt;b&gt;bold&lt;/b&gt; &amp; &#39;it&#39;"},
	}

	for _, tc := range tests {
		got := sanitize.EscapeHTML(tc.input)
		if got != tc.want {
			t.Errorf("EscapeHTML(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
