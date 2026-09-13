package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func resolveStringAlias(
	cmd *cobra.Command,
	canonicalName, canonicalValue, legacyName, legacyValue string,
) (value string, set bool, err error) {
	canonicalSet := cmd.Flags().Changed(canonicalName)
	legacySet := cmd.Flags().Changed(legacyName)
	if canonicalSet && legacySet {
		return "", false, fmt.Errorf("use only --%s or --%s", canonicalName, legacyName)
	}
	if canonicalSet {
		return canonicalValue, true, nil
	}
	if legacySet {
		return legacyValue, true, nil
	}
	return "", false, nil
}

func resolveIntAlias(
	cmd *cobra.Command,
	canonicalName string, canonicalValue int,
	legacyName string, legacyValue int,
) (value int, set bool, err error) {
	canonicalSet := cmd.Flags().Changed(canonicalName)
	legacySet := cmd.Flags().Changed(legacyName)
	if canonicalSet && legacySet {
		return 0, false, fmt.Errorf("use only --%s or --%s", canonicalName, legacyName)
	}
	if canonicalSet {
		return canonicalValue, true, nil
	}
	if legacySet {
		return legacyValue, true, nil
	}
	return 0, false, nil
}
