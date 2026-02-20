package model

import (
	"unicode/utf8"
)

const reasonPhraseMaxLen = 1024

type MoqtReasonPhrase string

func NewReasonPhrase(phrase string) MoqtReasonPhrase {
	if len(phrase) > reasonPhraseMaxLen {
		return ""
	}

	if !utf8.ValidString(phrase) {
		return ""
	}

	return MoqtReasonPhrase(phrase)
}
