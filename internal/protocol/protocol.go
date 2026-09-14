package protocol

import "errors"

const (
	CLIVersion      = "0.6.0"
	ProtocolVersion = "1"
)

const (
	CodeSuccess               = "OK"
	CodeValidationFailed      = "PRUMO_VALIDATION_FAILED"
	CodePolicyDenied          = "PRUMO_POLICY_DENIED"
	CodeConfiguration         = "PRUMO_CONFIGURATION_ERROR"
	CodeUnavailableCapability = "PRUMO_UNAVAILABLE_CAPABILITY"
	CodeInternal              = "PRUMO_INTERNAL_ERROR"
	CodeProjectNotFound       = "PRUMO_PROJECT_NOT_FOUND"
)

var (
	ErrProjectNotFound = errors.New("project root not found")
	ErrInvalidProject  = errors.New("invalid project manifest")
	ErrValidation      = errors.New("validation failed")
	ErrConfiguration   = errors.New("configuration error")
)

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Envelope struct {
	ProtocolVersion string       `json:"protocol_version"`
	Ok              bool         `json:"ok"`
	Data            any          `json:"data"`
	Diagnostics     []Diagnostic `json:"diagnostics"`
	Warnings        []string     `json:"warnings"`
}

func OkEnvelope(data any) Envelope {
	return Envelope{
		ProtocolVersion: ProtocolVersion,
		Ok:              true,
		Data:            data,
		Diagnostics:     []Diagnostic{},
		Warnings:        []string{},
	}
}

func ErrEnvelope(diags ...Diagnostic) Envelope {
	return Envelope{
		ProtocolVersion: ProtocolVersion,
		Ok:              false,
		Data:            struct{}{},
		Diagnostics:     diags,
		Warnings:        []string{},
	}
}
