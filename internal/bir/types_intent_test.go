package bir_test

import (
	"testing"

	"github.com/saero-ai/xcaffold/internal/bir"
	"github.com/stretchr/testify/assert"
)

func TestIntentType_ConstantsExist(t *testing.T) {
	assert.Equal(t, bir.IntentType("constraint"), bir.IntentConstraint)
	assert.Equal(t, bir.IntentType("procedure"), bir.IntentProcedure)
	assert.Equal(t, bir.IntentType("automation"), bir.IntentAutomation)
}

func TestIntentType_StringValues(t *testing.T) {
	assert.Equal(t, "constraint", string(bir.IntentConstraint))
	assert.Equal(t, "procedure", string(bir.IntentProcedure))
	assert.Equal(t, "automation", string(bir.IntentAutomation))
}

func TestFunctionalIntent_FieldsExist(t *testing.T) {
	fi := bir.FunctionalIntent{
		Type:    bir.IntentConstraint,
		Content: "NEVER do this",
		Source:  "NEVER keyword",
	}
	assert.Equal(t, bir.IntentConstraint, fi.Type)
	assert.Equal(t, "NEVER do this", fi.Content)
	assert.Equal(t, "NEVER keyword", fi.Source)
}

func TestSemanticUnit_HasIntentsField(t *testing.T) {
	unit := bir.SemanticUnit{
		ID:             "my-agent",
		SourceKind:     bir.SourceAgent,
		SourcePlatform: "claude",
		SourcePath:     "/path/to/file.md",
		ResolvedBody:   "some body",
		Intents: []bir.FunctionalIntent{
			{Type: bir.IntentConstraint, Content: "NEVER do this", Source: "NEVER keyword"},
		},
	}
	assert.Equal(t, "my-agent", unit.ID)
	assert.Equal(t, bir.SourceAgent, unit.SourceKind)
	assert.Equal(t, "claude", unit.SourcePlatform)
	assert.Equal(t, "/path/to/file.md", unit.SourcePath)
	assert.Equal(t, "some body", unit.ResolvedBody)
	assert.Len(t, unit.Intents, 1)
	assert.Equal(t, bir.IntentConstraint, unit.Intents[0].Type)
}

func TestSemanticUnit_EmptyIntentsByDefault(t *testing.T) {
	unit := bir.SemanticUnit{
		ID:         "empty",
		SourceKind: bir.SourceRule,
	}
	assert.Equal(t, "empty", unit.ID)
	assert.Equal(t, bir.SourceRule, unit.SourceKind)
	assert.Nil(t, unit.Intents)
}
