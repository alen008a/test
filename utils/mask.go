package utils

func GenerateNickname(str string) string {
	var buff []byte
	strLen := len(str)
	if strLen < 0 {
		return ""
	} else if strLen < 6 {
		buff = make([]byte, 5)
		copy(buff[:2], str[:2])
		copy(buff[2:], "***")
	} else {
		buff = make([]byte, 7)
		copy(buff[:2], str[:2])
		copy(buff[2:], "***")
		copy(buff[5:], str[len(str)-2:])
	}
	return string(buff)
}