package domain

// CloudProvider represents a supported cloud provider.
type CloudProvider string

const (
	ProviderAWS CloudProvider = "aws"
	ProviderGCP CloudProvider = "gcp"
)

// ValidProviders returns all supported cloud providers.
func ValidProviders() []CloudProvider {
	return []CloudProvider{ProviderAWS, ProviderGCP}
}

// IsValid checks whether the provider is a known supported provider.
func (p CloudProvider) IsValid() bool {
	switch p {
	case ProviderAWS, ProviderGCP:
		return true
	default:
		return false
	}
}

// String returns the string representation.
func (p CloudProvider) String() string {
	return string(p)
}
