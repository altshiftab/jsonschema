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

	"github.com/altshiftab/jsonschema/pkg/types"
	"golang.org/x/net/idna"
)

// hostnameFormat requires a valid hostname.
func hostnameFormat(instance any, state *types.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidHostname(s, false) {
		return fmt.Errorf("%q is not a valid hostname", s)
	}
	return nil
}

// idnHostnameFormat requires a valid internationalized hostname.
func idnHostnameFormat(instance any, state *types.ValidationState) error {
	s, ok := instance.(string)
	if !ok {
		return nil
	}
	if !isValidHostname(s, true) {
		return fmt.Errorf("%q is not a valid internationalized hostname", s)
	}
	return nil
}

// hostnameProfile returns the IDNA profile to use for
// non-internationalized hostnames.
var hostnameProfile = sync.OnceValue(func() *idna.Profile {
	return idna.New(idna.ValidateForRegistration())
})

// idnHostnameProfile returns the IDNA profile to use for
// internationalized hostnames.
var idnHostnameProfile = sync.OnceValue(func() *idna.Profile {
	return idna.New(
		idna.VerifyDNSLength(true),
		idna.ValidateLabels(true),
		idna.BidiRule(),
	)
})

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
		for i := range len(s) {
			if s[i]&0x80 != 0 {
				return false
			}
		}
	} else {
		// Permit all stops (RFC3490 section 3.1).
		s = strings.ReplaceAll(s, "\u3002", ".")
		s = strings.ReplaceAll(s, "\uff0e", ".")
		s = strings.ReplaceAll(s, "\uff61", ".")

		// Check for RFC5892 rules that the idna package
		// doesn't check.
		var last, nextMustBe rune
		var nextMustBeGreek bool
		for _, c := range s {
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
				// Disallowed rune.
				return false

			case '\u00b7':
				if last != '\u006c' {
					return false
				}
				nextMustBe = '\u006c'

			case '\u0375':
				nextMustBeGreek = true

			case '\u05f3', '\u05f4':
				if !unicode.Is(unicode.Hebrew, last) {
					return false
				}

			case '\u30fb':
				found := false
				for _, c := range s {
					if unicode.Is(unicode.Hiragana, c) || unicode.Is(unicode.Katakana, c) || unicode.Is(unicode.Han, c) {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}

			last = c
		}
		if nextMustBe != 0 || nextMustBeGreek {
			return false
		}
	}

	var p *idna.Profile
	if idn {
		// p = idnHostnameProfile()
		p = hostnameProfile()
	} else {
		p = hostnameProfile()
	}
	if _, err := p.ToASCII(s); err != nil {
		return false
	}

	return true
}
