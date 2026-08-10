// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package format

import (
	"fmt"
	"net/netip"
	"strings"
	"sync"
	"unicode"

	"github.com/altshiftab/jsonschema/pkg/schema"
	"golang.org/x/net/idna"
)

// hostnameFormat requires a valid hostname.
func hostnameFormat(instance any, state *schema.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidHostname(s, false) {
		return &schema.ValidationError{Message: fmt.Sprintf("%q is not a valid hostname", s)}
	}
	return nil
}

// idnHostnameFormat requires a valid internationalized hostname.
func idnHostnameFormat(instance any, state *schema.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidHostname(s, true) {
		return &schema.ValidationError{Message: fmt.Sprintf("%q is not a valid internationalized hostname", s)}
	}
	return nil
}

// hostnameProfile returns the IDNA profile to use for
// non-internationalized hostnames.
var hostnameProfile = sync.OnceValue(func() *idna.Profile {
	return idna.New(idna.ValidateForRegistration())
})

// acePrefix is the ASCII Compatible Encoding prefix (RFC 5890).
const acePrefix = "xn--"

// isValidHostname reports whether this is a valid hostname.
// If idn is true, this permits internationalized hostnames.
func isValidHostname(s string, idn bool) bool {
	if _, err := netip.ParseAddr(s); err == nil {
		// Valid IP address.
		return true
	}

	// Underscores are permitted by idna but not by the testsuite.
	if strings.Contains(s, "_") {
		return false
	}

	if !idn {
		if !isASCIIString(s) {
			return false
		}
	} else {
		// Permit all stops (RFC3490 section 3.1).
		s = strings.ReplaceAll(s, "\u3002", ".")
		s = strings.ReplaceAll(s, "\uff0e", ".")
		s = strings.ReplaceAll(s, "\uff61", ".")
	}

	// An empty root label (trailing dot) is not permitted by the testsuite.
	if strings.HasSuffix(s, ".") {
		return false
	}

	for label := range strings.SplitSeq(s, ".") {
		if hasACEPrefix(label) {
			// An A-label must be valid Punycode and must decode to a
			// non-ASCII U-label that satisfies the RFC 5892 rules
			// (RFC 5891 section 4.2.3.1).
			decoded, err := idna.Punycode.ToUnicode(acePrefix + label[len(acePrefix):])
			if err != nil {
				return false
			}
			if isASCIIString(decoded) || !isValidUnicodeLabel(decoded) {
				return false
			}
		} else if idn && !isValidUnicodeLabel(label) {
			return false
		}
	}

	// The registration profile matches the JSON schema testsuite
	// more closely than an idna.Profile with VerifyDNSLength,
	// ValidateLabels, and BidiRule, also for internationalized
	// hostnames, given the checks performed above.
	if _, err := hostnameProfile().ToASCII(s); err != nil {
		return false
	}

	return true
}

// isASCIIString reports whether s consists solely of ASCII bytes.
func isASCIIString(s string) bool {
	for i := range len(s) {
		if s[i]&0x80 != 0 {
			return false
		}
	}
	return true
}

// hasACEPrefix reports whether the label starts with the ASCII
// Compatible Encoding prefix "xn--", case-insensitively.
func hasACEPrefix(label string) bool {
	return len(label) >= len(acePrefix) &&
		(label[0] == 'x' || label[0] == 'X') &&
		(label[1] == 'n' || label[1] == 'N') &&
		label[2] == '-' && label[3] == '-'
}

// isValidUnicodeLabel checks a Unicode (U-label) hostname label for
// the RFC 5892 rules that the idna package doesn't check.
func isValidUnicodeLabel(label string) bool {
	var last, nextMustBe rune
	var nextMustBeGreek bool
	for _, c := range label {
		if nextMustBe != 0 && nextMustBe != c {
			return false
		}
		nextMustBe = 0

		if nextMustBeGreek {
			if !unicode.Is(unicode.Greek, c) {
				return false
			}
		}
		nextMustBeGreek = false

		switch c {
		case '\u0640', '\u07fa', '\u302e', '\u302f',
			'\u3031', '\u3032', '\u3033', '\u3034',
			'\u3035', '\u303b':
			// Disallowed rune (RFC 5892 section 2.6).
			return false

		case '\u00b7':
			// MIDDLE DOT must be surrounded by 'l'
			// (RFC 5892 appendix A.3).
			if last != '\u006c' {
				return false
			}
			nextMustBe = '\u006c'

		case '\u0375':
			// GREEK LOWER NUMERAL SIGN must be followed by Greek
			// (RFC 5892 appendix A.4).
			nextMustBeGreek = true

		case '\u05f3', '\u05f4':
			// HEBREW GERESH and GERSHAYIM must be preceded by Hebrew
			// (RFC 5892 appendixes A.5 and A.6).
			if !unicode.Is(unicode.Hebrew, last) {
				return false
			}

		case '\u30fb':
			// KATAKANA MIDDLE DOT requires Hiragana, Katakana, or Han
			// in the label (RFC 5892 appendix A.7).
			found := false
			for _, r := range label {
				if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Han, r) {
					found = true
					break
				}
			}
			if !found {
				return false
			}

		case '-', '\u200c', '\u200d', '\u06fd', '\u06fe', '\u0f0b', '\u3007':
			// Permitted: hyphen, the joiners (whose contextual rules
			// the idna profile checks), and the PVALID exceptions with
			// non-letter general categories (RFC 5892 section 2.6).

		default:
			// Everything else must be a letter, mark, decimal digit,
			// or letterlike number to be PVALID (RFC 5892 section 2);
			// symbols and punctuation are disallowed.
			if !unicode.In(c, unicode.L, unicode.M, unicode.Nd, unicode.Nl) {
				return false
			}
		}

		last = c
	}
	if nextMustBe != 0 || nextMustBeGreek {
		return false
	}
	return true
}
