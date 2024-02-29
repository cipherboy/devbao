package bao

func PrefixedAppend(slice []string, prefix string, values ...string) []string {
	for _, item := range values {
		slice = append(slice, prefix+item)
	}
	return slice
}
