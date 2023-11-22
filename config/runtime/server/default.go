package server

import "github.com/avenga/couper/config"

// SortDefault ensures that default roundtrips are the last item to be able to pipe the origin response body.
func SortDefault[V any](m map[string]V) []string {
	idx := -1

	var trips []string
	for k := range m {
		trips = append(trips, k)
	}

	for i, t := range trips {
		if t == config.DefaultNameLabel {
			idx = i
			break
		}
	}
	edx := len(trips) - 1
	if idx < 0 || edx == 0 || idx == edx {
		return trips
	}

	trips[idx], trips[edx] = trips[edx], trips[idx] // swap

	return trips
}
