package dictionary

const (
	SystemDictVersion = 0x7366d3f18bd111e7
	UserDictVersion   = 0xa50f31188bd211e7
	UserDictVersion2  = 0x9fdeb5a90168d868
)

func IsUserDictionary(version uint64) bool {
	return version == UserDictVersion || version == UserDictVersion2
}
