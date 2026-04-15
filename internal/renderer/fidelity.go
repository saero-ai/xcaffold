package renderer

// FidelityLevel categorizes the severity of a fidelity gap.
type FidelityLevel string

const (
	// LevelInfo indicates a lossless transformation or noteworthy mapping decision.
	LevelInfo FidelityLevel = "info"

	// LevelWarning indicates information loss. The output is still usable but
	// differs semantically from the source. Promoted to error in --fidelity strict mode.
	LevelWarning FidelityLevel = "warning"

	// LevelError indicates a condition severe enough to flag as an error even in
	// warn mode. Always promoted regardless of --fidelity setting.
	LevelError FidelityLevel = "error"
)

// FidelityNote describes a non-fatal information loss or transformation that
// occurred during rendering. Notes are returned alongside compiled output and
// are never a substitute for a compilation error.
type FidelityNote struct {
	Level      FidelityLevel `json:"level"                yaml:"level"`
	Target     string        `json:"target"               yaml:"target"`
	Kind       string        `json:"kind"                 yaml:"kind"`
	Resource   string        `json:"resource"             yaml:"resource"`
	Field      string        `json:"field,omitempty"      yaml:"field,omitempty"`
	Reason     string        `json:"reason"               yaml:"reason"`
	Code       string        `json:"code"                 yaml:"code"`
	Mitigation string        `json:"mitigation,omitempty" yaml:"mitigation,omitempty"`
}

// NewNote constructs a FidelityNote from positional arguments to reduce
// construction boilerplate inside renderer packages.
func NewNote(level FidelityLevel, target, kind, resource, field, code, reason, mitigation string) FidelityNote {
	return FidelityNote{
		Level:      level,
		Target:     target,
		Kind:       kind,
		Resource:   resource,
		Field:      field,
		Code:       code,
		Reason:     reason,
		Mitigation: mitigation,
	}
}
