package p

// old
type AnonField struct {
	Inner struct {
		X int
	}
}

// new
type AnonField struct {
	// i AnonField.Inner: removed
}
