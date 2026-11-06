package app

import (
	"context"
	"log"
	"sort"

	"earthmanlabs/resumaker/internal/store"
)

// FeatureGemini gates Tailor with Gemini. It is switched on per user with the
// admin command.
const FeatureGemini = "gemini"

// features lists every per-user feature that can be switched on, with a label
// for the admin command's output.
var features = map[string]string{
	FeatureGemini: "Tailor with Gemini",
}

// ValidFeature reports whether a feature name is one the app knows about.
func ValidFeature(name string) bool {
	_, ok := features[name]
	return ok
}

// FeatureNames returns the known feature names in sorted order.
func FeatureNames() []string {
	names := make([]string, 0, len(features))
	for name := range features {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FeatureEnabled reports whether a feature is on for a user.
func (s *Service) FeatureEnabled(ctx context.Context, user store.User, feature string) (bool, error) {
	return s.store.HasFeature(ctx, user.ID, feature)
}

// AIAvailableFor reports whether the user can use Tailor with Gemini: a key is
// configured on the server and the user's feature flag is on. A failed lookup
// is logged and treated as off, so the page still renders.
func (s *Service) AIAvailableFor(ctx context.Context, user store.User) bool {
	if s == nil || s.generator == nil {
		return false
	}
	enabled, err := s.FeatureEnabled(ctx, user, FeatureGemini)
	if err != nil {
		log.Printf("feature lookup for user %s: %v", user.ID, err)
		return false
	}
	return enabled
}
