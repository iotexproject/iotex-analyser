package main

import (
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/v2/action"
)

func appendIfMissing(slice []string, s string) []string {
	for _, element := range slice {
		if element == s {
			return slice
		}
	}
	return append(slice, s)
}

// getActionTypeString delegates to the shared implementation so this plugin
// and block_action can never disagree about what an action is called.
func getActionTypeString(act action.Action) string {
	return kernel.ActionTypeString(act)
}
