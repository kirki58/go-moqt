package model

import (
	"fmt"
	"unicode/utf8"
)

const reasonPhraseMaxLen = 1024

type MoqtReasonPhrase string

func NewReasonPhrase(phrase string) MoqtReasonPhrase {
	if len(phrase) > reasonPhraseMaxLen {
		panic(fmt.Sprintf("Reason Phrase must not exceed %d bytes", reasonPhraseMaxLen))
	}

	if !utf8.ValidString(phrase) {
		panic("Reason Phrase must be valid UTF-8 string")
	}

	return MoqtReasonPhrase(phrase)
}
