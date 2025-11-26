package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewScope_Validate(t *testing.T) {
	tests := []struct {
		name    string
		scope   ReviewScope
		wantErr bool
		errType error
	}{
		{
			name: "valid workload scope",
			scope: ReviewScope{
				Level: ScopeLevelWorkload,
			},
			wantErr: false,
		},
		{
			name: "valid pillar scope",
			scope: ReviewScope{
				Level:  ScopeLevelPillar,
				Pillar: ptrToPillar(PillarSecurity),
			},
			wantErr: false,
		},
		{
			name: "valid question scope",
			scope: ReviewScope{
				Level:      ScopeLevelQuestion,
				QuestionID: "sec_data_classification_1",
			},
			wantErr: false,
		},
		{
			name: "pillar scope missing pillar",
			scope: ReviewScope{
				Level: ScopeLevelPillar,
			},
			wantErr: true,
			errType: ErrPillarRequired,
		},
		{
			name: "question scope missing question ID",
			scope: ReviewScope{
				Level: ScopeLevelQuestion,
			},
			wantErr: true,
			errType: ErrQuestionIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scope.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func ptrToPillar(p Pillar) *Pillar {
	return &p
}
