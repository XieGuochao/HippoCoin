package host

// Signable ...
// The interface for all signable objects.
// Block should follow this interface.
type Signable interface {
	Hash() string
	Signature() string
	PublicKey() string
	CheckSignature() bool
	Check() bool
}
