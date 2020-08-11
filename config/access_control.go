package config

// AccessControl holds all active and inactive access control references.
type AccessControl struct {
	AccessControl        []string
	DisableAccessControl []string
}

// NewAccessControl creates the container object for ac configuration.
func NewAccessControl(ac, dac []string) AccessControl {
	return AccessControl{
		AccessControl:        ac,
		DisableAccessControl: dac,
	}
}

// List returns all active access controls.
func (ac AccessControl) List() []string {
	var result []string
	for _, c := range ac.AccessControl {
		if contains(ac.DisableAccessControl, c) != -1 {
			continue
		}
		result = append(result, c)
	}
	return result
}

// Merge appends control references in order.
func (ac AccessControl) Merge(oac AccessControl) AccessControl {
	for _, other := range oac.AccessControl {
		if contains(ac.AccessControl, other) != -1 {
			continue
		}
		ac.AccessControl = append(ac.AccessControl, other)
	}
	for _, other := range oac.DisableAccessControl {
		if contains(ac.DisableAccessControl, other) != -1 {
			continue
		}
		ac.DisableAccessControl = append(ac.DisableAccessControl, other)
	}
	return ac
}

func contains(a []string, b string) int {
	idx := -1
	for i := range a {
		if a[i] == b {
			return i
		}
	}
	return idx
}
