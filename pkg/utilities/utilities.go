package utilities

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"strconv"
	"voda_parser/pkg/logger"
)

// ToJSON - преобразует объект в JSON
func ToJSON(object any) string {
	jsonByte, err := json.Marshal(object)
	if err != nil {
		logger.Error("Ошибка при получении JSON: ", err.Error())
		return ""
	}

	var out bytes.Buffer
	err = json.Indent(&out, jsonByte, "", "   ") // Форматируем jsonByte
	if err != nil {
		return string(jsonByte) // Возвращаем неформатированный JSON в случае ошибки
	}
	return out.String()
}

// StrToUint - Конвертирует строку в uint
func StrToUint(s string) uint {
	i, err := strconv.Atoi(s)
	if err != nil {
		logger.Error("%s", err.Error())
		return 0
	}
	return uint(i)
}

// RandInt - возвращает случайное число от min до max
func RandInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

// GenerateRandomString - генерирует случайный набор символов (англ алфавит, case uppercase + символ _ и цифры от 0 до 9)
func GenerateRandomString(length int) string {
	const alphabet = "QOS4rT08Dm7dZVOPwucfM2haFiNyEjBK3UtC9IqY_lzv6XpWgWsAJebG5H1RxnLbK"

	var result = make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(result)
}
