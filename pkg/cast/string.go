package cast

func ToSlicePtrString(s []string) []*string {
	out := make([]*string, len(s))

	for i, v := range s {
		v := v
		out[i] = &v
	}

	return out
}
