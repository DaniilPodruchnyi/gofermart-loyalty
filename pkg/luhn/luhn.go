package luhn

// Validate проверяет номер по алгоритму Луна
func Validate(number string) bool {
	if len(number) == 0 {
		return false
	}

	sum := 0
	isSecond := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')

		if digit < 0 || digit > 9 {
			return false
		}

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}
