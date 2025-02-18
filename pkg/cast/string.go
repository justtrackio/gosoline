package cast

func ToSlicePtrString(s []string) []*string {
	out := make([]*string, len(s))

	for i, v := range s {
		out[i] = &v
	}

	return out
}
