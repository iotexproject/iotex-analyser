package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/iotexproject/iotex-core/action"
)

func firstLowerCase(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func getActionTypeString(action action.Action) string {
	actionType := fmt.Sprintf("%T", action)
	return firstLowerCase(strings.TrimLeft(actionType, "*action."))
}
