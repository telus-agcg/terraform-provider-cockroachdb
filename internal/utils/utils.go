package utils

import (
	"context"
	"fmt"
	"strings"
)

func SliceContainsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func ToPtr[T any](val T) *T {
	return &val
}

func IsNilOrEmpty(str *string) bool {
	if str == nil {
		return true
	}
	if *str == "" {
		return true
	}
	if strings.Trim(*str, " ") == "" {
		return true
	}
	return false
}

var allowedPrivileges = map[string][]string{
	"database": {"ALL", "CREATE", "CONNECT", "TEMPORARY"},
	"table":    {"ALL", "SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"},
	"schema":   {"ALL", "CREATE", "USAGE"},
}

func ValidatePrivileges(ctx context.Context, objectType string, privileges []string) error {
	allowed, ok := allowedPrivileges[objectType]
	if !ok {
		return fmt.Errorf("unknown object type %s", objectType)
	}

	for _, priv := range privileges {
		if !SliceContainsStr(allowed, priv) {
			return fmt.Errorf("%s is not an allowed privilege for object type %s", priv, objectType)
		}
	}
	return nil
}
