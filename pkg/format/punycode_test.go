package format

import "testing"

func TestDecodePunycode(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		encoded string
		want    string
		wantOK  bool
	}{
		{name: "german umlaut", encoded: "bcher-kva", want: "bücher", wantOK: true},
		{name: "all basic with trailing delimiter", encoded: "example-", want: "example", wantOK: true},
		{name: "empty", encoded: "", want: "", wantOK: true},
		{name: "short with umlaut", encoded: "tst-qla", want: "täst", wantOK: true},
		{name: "no basic code points", encoded: "wgv71a119e", want: "日本語", wantOK: true}, //nolint:gosmopolitan // Intentional Unicode test data.
		{name: "hangul", encoded: "9t4b11yi5a", want: "테스트", wantOK: true},
		{name: "hangul with tone mark", encoded: "07jt112bpxg", want: "\uc2e4\u302e\ub840", wantOK: true},
		{name: "extended only", encoded: "abc", want: "\u0082\u0081\u0080", wantOK: true},
		{name: "invalid digit", encoded: "invalid!", wantOK: false},
		{name: "non-ASCII input", encoded: "bücher", wantOK: false},
		{name: "truncated extended part", encoded: "bcher-k", wantOK: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, ok := decodePunycode(testCase.encoded)
			if ok != testCase.wantOK {
				t.Fatalf("decodePunycode(%q) ok = %t, want %t", testCase.encoded, ok, testCase.wantOK)
			}
			if ok && got != testCase.want {
				t.Errorf("decodePunycode(%q) = %q, want %q", testCase.encoded, got, testCase.want)
			}
		})
	}
}

func TestEncodePunycode(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		label  string
		want   string
		wantOK bool
	}{
		{name: "german umlaut", label: "bücher", want: "bcher-kva", wantOK: true},
		{name: "short with umlaut", label: "täst", want: "tst-qla", wantOK: true},
		{name: "no basic code points", label: "日本語", want: "wgv71a119e", wantOK: true}, //nolint:gosmopolitan // Intentional Unicode test data.
		{name: "hangul", label: "테스트", want: "9t4b11yi5a", wantOK: true},
		{name: "all ASCII", label: "example", want: "example-", wantOK: true},
		{name: "empty", label: "", want: "", wantOK: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, ok := encodePunycode(testCase.label)
			if ok != testCase.wantOK {
				t.Fatalf("encodePunycode(%q) ok = %t, want %t", testCase.label, ok, testCase.wantOK)
			}
			if ok && got != testCase.want {
				t.Errorf("encodePunycode(%q) = %q, want %q", testCase.label, got, testCase.want)
			}
		})
	}
}

func TestPunycodeRoundTrip(t *testing.T) {
	t.Parallel()
	testCases := []string{"bücher", "täst", "日本語", "테스트", "\uc2e4\u302e\ub840", "münchen-3ya-mix", "ですat"} //nolint:gosmopolitan // Intentional Unicode test data.
	for _, label := range testCases {
		t.Run(label, func(t *testing.T) {
			t.Parallel()
			encoded, ok := encodePunycode(label)
			if !ok {
				t.Fatalf("encodePunycode(%q) failed", label)
			}
			decoded, ok := decodePunycode(encoded)
			if !ok {
				t.Fatalf("decodePunycode(%q) failed", encoded)
			}
			if decoded != label {
				t.Errorf("round trip of %q via %q = %q", label, encoded, decoded)
			}
		})
	}
}
