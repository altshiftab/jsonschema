// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package format

import (
	"math"
	"slices"
	"strings"
	"unicode/utf8"
)

// This file implements the Punycode algorithm of RFC 3492,
// as used for the A-labels of internationalized hostnames (RFC 5890).

const (
	punyBase        = 36
	punyTMin        = 1
	punyTMax        = 26
	punySkew        = 38
	punyDamp        = 700
	punyInitialBias = 72
	punyInitialN    = 128
	punyDelimiter   = '-'
)

// punyAdapt is the bias adaptation function (RFC 3492 section 6.1).
func punyAdapt(delta, numPoints int, firstTime bool) int {
	if firstTime {
		delta /= punyDamp
	} else {
		delta /= 2
	}
	delta += delta / numPoints
	k := 0
	for delta > ((punyBase-punyTMin)*punyTMax)/2 {
		delta /= punyBase - punyTMin
		k += punyBase
	}
	return k + (punyBase-punyTMin+1)*delta/(delta+punySkew)
}

// punyDigitValue returns the numeric value of a Punycode digit.
func punyDigitValue(b byte) (int, bool) {
	switch {
	case 'a' <= b && b <= 'z':
		return int(b - 'a'), true
	case 'A' <= b && b <= 'Z':
		return int(b - 'A'), true
	case '0' <= b && b <= '9':
		return int(b-'0') + punyTMax, true
	default:
		return 0, false
	}
}

// punyEncodeDigit returns the (lowercase) Punycode digit for a value.
func punyEncodeDigit(d int) byte {
	if d < punyTMax {
		//nolint:gosec // d is a Punycode digit value in [0, 35].
		return byte('a' + d)
	}
	//nolint:gosec // d is a Punycode digit value in [0, 35].
	return byte('0' + d - punyTMax)
}

// punyThreshold returns the threshold for digit position k (RFC 3492
// section 6.2 and 6.3).
func punyThreshold(k, bias int) int {
	switch {
	case k <= bias:
		return punyTMin
	case k >= bias+punyTMax:
		return punyTMax
	default:
		return k - bias
	}
}

// decodePunycode decodes the Punycode of an A-label, without the
// "xn--" ACE prefix, and reports whether the encoding was valid.
func decodePunycode(encoded string) (string, bool) {
	var output []rune
	pos := 0
	if idx := strings.LastIndexByte(encoded, punyDelimiter); idx >= 0 {
		for _, b := range []byte(encoded[:idx]) {
			if b >= utf8.RuneSelf {
				return "", false
			}
			output = append(output, rune(b))
		}
		pos = idx + 1
	}

	i, n, bias := 0, punyInitialN, punyInitialBias
	for pos < len(encoded) {
		oldI, w := i, 1
		for k := punyBase; ; k += punyBase {
			if pos >= len(encoded) {
				return "", false
			}
			d, ok := punyDigitValue(encoded[pos])
			if !ok {
				return "", false
			}
			pos++
			if w == 0 || d > (math.MaxInt32-i)/w {
				return "", false
			}
			i += d * w
			t := punyThreshold(k, bias)
			if d < t {
				break
			}
			if w > math.MaxInt32/(punyBase-t) {
				return "", false
			}
			w *= punyBase - t
		}
		x := len(output) + 1
		bias = punyAdapt(i-oldI, x, oldI == 0)
		if i/x > math.MaxInt32-n {
			return "", false
		}
		n += i / x
		i %= x
		if n > utf8.MaxRune || (0xd800 <= n && n <= 0xdfff) {
			// Out of range or a surrogate.
			return "", false
		}
		//nolint:gosec // n is at most utf8.MaxRune, checked above.
		output = slices.Insert(output, i, rune(n))
		i++
	}

	return string(output), true
}

// encodePunycode encodes a label as Punycode, without the "xn--"
// ACE prefix, and reports whether encoding succeeded.
func encodePunycode(label string) (string, bool) {
	runes := []rune(label)
	var out []byte
	for _, r := range runes {
		if r < utf8.RuneSelf {
			out = append(out, byte(r))
		}
	}
	basic := len(out)
	handled := basic
	if basic > 0 {
		out = append(out, punyDelimiter)
	}

	n, delta, bias := punyInitialN, 0, punyInitialBias
	for handled < len(runes) {
		m := int(utf8.MaxRune) + 1
		for _, r := range runes {
			if int(r) >= n && int(r) < m {
				m = int(r)
			}
		}
		if m-n > (math.MaxInt32-delta)/(handled+1) {
			return "", false
		}
		delta += (m - n) * (handled + 1)
		n = m
		for _, r := range runes {
			c := int(r)
			if c < n {
				delta++
				if delta > math.MaxInt32 {
					return "", false
				}
				continue
			}
			if c > n {
				continue
			}
			q := delta
			for k := punyBase; ; k += punyBase {
				t := punyThreshold(k, bias)
				if q < t {
					break
				}
				out = append(out, punyEncodeDigit(t+(q-t)%(punyBase-t)))
				q = (q - t) / (punyBase - t)
			}
			out = append(out, punyEncodeDigit(q))
			bias = punyAdapt(delta, handled+1, handled == basic)
			delta = 0
			handled++
		}
		delta++
		n++
	}

	return string(out), true
}
