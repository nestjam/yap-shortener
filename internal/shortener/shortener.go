package shortener

const alphabet = "ynAJfoSgdXHB5VasEMtcbPCr1uNZ4LG723ehWkvwYR6KpxjTm8iQUFqz9D"

var alphabetLen = uint32(len(alphabet))

// Shorten конвертирует число в строку на основании алфавита из 58 символов.
func Shorten(id uint32) string {
	letters := []byte{}

	for {
		letters = append(letters, alphabet[id%alphabetLen])
		id /= alphabetLen

		if id == 0 {
			break
		}
	}

	return string(letters)
}
