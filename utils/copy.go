package utils

func Clone(b []byte) []byte {
	return append([]byte(nil), b...)
}
