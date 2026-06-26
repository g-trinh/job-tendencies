package kernel

import "fmt"

// RemotePolicy describes the remote-work policy advertised by a job listing.
type RemotePolicy string

const (
	// RemotePolicyOnSite requires full on-site presence (présentiel).
	RemotePolicyOnSite RemotePolicy = "on_site"
	// RemotePolicyHybrid allows a mix of on-site and remote work.
	RemotePolicyHybrid RemotePolicy = "hybrid"
	// RemotePolicyFullRemote allows fully remote work (télétravail complet).
	RemotePolicyFullRemote RemotePolicy = "full_remote"
)

var validRemotePolicies = map[RemotePolicy]bool{
	RemotePolicyOnSite:     true,
	RemotePolicyHybrid:     true,
	RemotePolicyFullRemote: true,
}

// ParseRemotePolicy parses a RemotePolicy from a raw string, returning an error
// if the value is not recognised.
func ParseRemotePolicy(s string) (RemotePolicy, error) {
	rp := RemotePolicy(s)
	if !validRemotePolicies[rp] {
		return "", fmt.Errorf("unknown remote policy %q; valid values: on_site, hybrid, full_remote", s)
	}
	return rp, nil
}

// IsValid reports whether r is a known RemotePolicy value.
func (r RemotePolicy) IsValid() bool { return validRemotePolicies[r] }
