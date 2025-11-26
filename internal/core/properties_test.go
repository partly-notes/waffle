package core

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Property test generators for Waffle types

// genSessionID generates valid session IDs (UUID format)
func genSessionID() gopter.Gen {
	return gen.RegexMatch(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
}

// genPillar generates valid pillar values
func genPillar() gopter.Gen {
	return gen.OneConstOf(
		PillarOperationalExcellence,
		PillarSecurity,
		PillarReliability,
		PillarPerformanceEfficiency,
		PillarCostOptimization,
		PillarSustainability,
	)
}

// genScopeLevel generates valid scope levels
func genScopeLevel() gopter.Gen {
	return gen.OneConstOf(
		ScopeLevelWorkload,
		ScopeLevelPillar,
		ScopeLevelQuestion,
	)
}

// genReviewScope generates valid review scopes
func genReviewScope() gopter.Gen {
	return gen.OneGenOf(
		// Workload scope
		gen.Const(ReviewScope{Level: ScopeLevelWorkload}),
		// Pillar scope
		genPillar().Map(func(p Pillar) ReviewScope {
			return ReviewScope{
				Level:  ScopeLevelPillar,
				Pillar: &p,
			}
		}),
		// Question scope
		gen.Identifier().Map(func(qid string) ReviewScope {
			return ReviewScope{
				Level:      ScopeLevelQuestion,
				QuestionID: qid,
			}
		}),
	)
}

// genSessionStatus generates valid session statuses
func genSessionStatus() gopter.Gen {
	return gen.OneConstOf(
		SessionStatusCreated,
		SessionStatusInProgress,
		SessionStatusCompleted,
		SessionStatusFailed,
	)
}

// genConfidenceScore generates valid confidence scores (0.0 to 1.0)
func genConfidenceScore() gopter.Gen {
	return gen.Float64Range(0.0, 1.0)
}

// TestProperty_ReviewScopeValidation tests that all generated review scopes are valid
func TestProperty_ReviewScopeValidation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("all generated review scopes should be valid", prop.ForAll(
		func(scope ReviewScope) bool {
			err := scope.Validate()
			return err == nil
		},
		genReviewScope(),
	))

	properties.TestingRun(t)
}

// TestProperty_ConfidenceScoreBounds tests that confidence scores are always in valid range
func TestProperty_ConfidenceScoreBounds(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("confidence scores should be between 0.0 and 1.0", prop.ForAll(
		func(score float64) bool {
			return score >= 0.0 && score <= 1.0
		},
		genConfidenceScore(),
	))

	properties.TestingRun(t)
}

// Example of how to use these generators in actual property tests
// This will be expanded in later tasks when implementing actual functionality
func TestProperty_Example(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("example property test", prop.ForAll(
		func(sessionID string, workloadID string, scope ReviewScope) bool {
			// This is a placeholder - actual tests will be implemented in later tasks
			// For now, just verify that generated values are non-empty
			return sessionID != "" && workloadID != "" && scope.Level >= 0
		},
		genSessionID(),
		gen.Identifier(),
		genReviewScope(),
	))

	properties.TestingRun(t)
}
