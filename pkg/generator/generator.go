package generator

import (
	"encoding/csv"
	"fmt"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"os"
	"path/filepath"
	"time"
	"voda_parser/pkg/types"
)

func GenerateFile(catalogs []*types.Catalog, chLog chan string) {
	// Данные для CSV файла
	data := [][]string{
		{"Товар", "vodaplus.ru", "Ссылка на товар"},
	}

	// Добавляем данные о товарах в массив
	for _, catalog := range catalogs {
		for _, product := range catalog.Products {
			data = append(data, []string{product.Name, fmt.Sprintf("%s", product.Price), product.URL})
		}
	}

	pwd, _ := os.Getwd()
	fileName := "data.csv"
	path := filepath.Join(pwd, fileName)
	// Открытие файла для записи
	file, err := os.Create(path)
	if err != nil {
		chLog <- fmt.Sprintf("Ошибка открытия файла: %v", err)
		return
	}
	defer file.Close()

	decoder := unicode.UTF8.NewDecoder()
	t := transform.NewWriter(file, decoder)

	// Создаем CSV-писателя с кодировкой Windows-1251
	writer := csv.NewWriter(t)
	defer writer.Flush()

	// Запись данных в файл
	for _, record := range data {
		if err := writer.Write(record); err != nil {
			chLog <- fmt.Sprintf("Ошибка записи в файл: %v", err)
		}
	}

	chLog <- fmt.Sprintf("Таблица сгенерирована по адресу: %s", path)
	time.Sleep(time.Second * 10)
	chLog <- "stop"
}
