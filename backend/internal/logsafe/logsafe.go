// Package logsafe renders untrusted values safe to embed in log records.
package logsafe

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxLen bounds a single logged field so one caller cannot flood a log line.
const maxLen = 256

// truncationSuffix marks a value shortened by Clean.
const truncationSuffix = "...(truncated)"

// Clean makes s safe to write into a log record.
//
// CR and LF become spaces, then every remaining Unicode control character
// (C0, C1, DEL) and format character is dropped. That covers line forgery,
// ESC-initiated ANSI/OSC terminal escape sequences, NUL truncation,
// U+2028/U+2029 line separators, and the bidi overrides used to spoof how a
// line reads. The result is truncated to maxLen bytes on a rune boundary.
//
// Use Clean for values reaching fmt-style sinks (log.Printf, gin's access
// logger), which perform no escaping of their own. It is also applied to slog
// attributes in this repo: slog's TextHandler and JSONHandler already quote
// values via strconv.Quote, so there Clean is defense-in-depth against a future
// custom handler that does not.
//
// Do not "simplify" the two strings.ReplaceAll calls away. The control-character
// filter below subsumes them, but they are the only construct CodeQL's
// go/log-injection query recognizes as a barrier: its sanitizer matches a
// replace whose replaced string is "\r" or "\n". Notably regexp.ReplaceAllString
// is NOT recognized -- an earlier fix in this repo used it and the alerts
// stayed open -- and neither is a regex validation guard, since the query
// defines no barrier guard and a validated-but-unmodified value stays tainted.
// There must also be no early return: any path from parameter to result that
// skips a ReplaceAll call reopens the alerts.
//
// The replacement string does not affect the barrier, so a space is used rather
// than "" to avoid gluing tokens together across a stripped newline.
func Clean(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	s = strings.Map(func(r rune) rune {
		switch {
		case r == utf8.RuneError:
			return -1
		case r == '\u2028' || r == '\u2029':
			return -1
		case unicode.IsControl(r):
			return -1
		case unicode.Is(unicode.Cf, r):
			return -1
		}
		return r
	}, s)

	if len(s) > maxLen {
		cut := maxLen
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + truncationSuffix
	}
	return s
}

// CleanErr applies Clean to err.Error(), returning "" for a nil error.
// Prefer it over logging an error value directly when the error may wrap
// upstream- or user-controlled text such as URLs, checksums, or filenames.
func CleanErr(err error) string {
	if err == nil {
		return ""
	}
	return Clean(err.Error())
}
