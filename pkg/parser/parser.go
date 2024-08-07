package parser

import (
	"fmt"
	"github.com/antchfx/htmlquery"
	"strings"
	"sync"
	"time"
	"voda_parser/pkg/generator"
	"voda_parser/pkg/logger"
	"voda_parser/pkg/types"
	"voda_parser/pkg/utilities"
)

func workerCatalogs(catalogs []*types.Catalog, ch chan map[int]*types.Product, mu *sync.Mutex, wg *sync.WaitGroup) {
	logger.Info("Запустили воркер")
	for {
		select {
		case product, ok := <-ch:
			if !ok {
				logger.Error("Канал воркера каталогов закрыт")
				return
			}

			for index, pr := range product {
				mu.Lock()
				catalogs[index].Products = append(catalogs[index].Products, pr)
				mu.Unlock()
			}
		case <-time.After(time.Second * 2):
			wg.Done()
			return
		}
	}
}

func workerPrices(ch chan map[*types.Product]string, mu *sync.Mutex, wg *sync.WaitGroup) {
	logger.Info("Запустили воркер цен")
	for {
		select {
		case mp, ok := <-ch:
			if !ok {
				logger.Error("Канал воркера цен закрыт")
				return
			}

			for product, p := range mp {
				mu.Lock()
				product.Price = p
				mu.Unlock()
			}
		case <-time.After(time.Minute * 2):
			wg.Done()
			return
		}
	}
}

func StartParse(chLog chan string) {
	chLog <- fmt.Sprintf("Запуск программы...")
	chLog <- fmt.Sprintf("Подключение к %s ...", types.URL)
	_, err := htmlquery.LoadURL(types.CATALOG_URL)
	if err != nil {
		chLog <- fmt.Sprintf("Ошибка при подключении к %s: %v", types.URL, err)
		return
	}
	chLog <- fmt.Sprintf("Успешное подключение, загружаем каталоги...")
	catalogs, err := LoadCatalogs()
	if err != nil {
		logger.Error("Ошибка при загрузке каталогов: %v", err)
		return
	}

	chLog <- fmt.Sprintf("Каталоги успешно загружены, количество каталогов: %d", len(catalogs))

	chProduct := make(chan map[int]*types.Product)
	muCt := sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(1)
	go workerCatalogs(catalogs, chProduct, &muCt, &wg)

	//Загружаем товары в каталоги
	{
		chLog <- fmt.Sprintf("Загрузка товаров из каталогов...")
		for index, catalog := range catalogs {
			wg.Add(1)
			go LoadProduct(index, catalog.URL, chProduct, &wg, nil)
		}

		wg.Wait()
		close(chProduct)
	}

	productsCount := 0
	for _, catalog := range catalogs {
		logger.Info("В каталоге %s, %d товаров", catalog.Name, len(catalog.Products))
		productsCount += len(catalog.Products)
	}

	chLog <- fmt.Sprintf("Товары успешно загружены, количество товаров: %d", productsCount)
	chLog <- fmt.Sprintf("Загружаем статистику по товарам...")

	chPrices := make(chan map[*types.Product]string)
	muPrice := sync.Mutex{}
	var wgPrice sync.WaitGroup
	wgPrice.Add(1)
	go workerPrices(chPrices, &muPrice, &wgPrice)

	var wgWaiter sync.WaitGroup
	//Загружаем цены в товары
	{
		for _, catalog := range catalogs {
			wgWaiter.Add(1)
			go func() {
				defer wgWaiter.Done()
				for _, product := range catalog.Products {
					wgPrice.Add(1)
					go LoadPrice(product, chPrices, &wgPrice, nil)
				}
			}()
		}

		wgWaiter.Wait()
		wgPrice.Wait()
		close(chPrices)
	}

	pricesBad := 0
	var badLink []string
	for _, catalog := range catalogs {
		for _, product := range catalog.Products {
			if len(product.Price) == 0 {
				pricesBad++
				badLink = append(badLink, product.URL)
			}
		}
	}

	logger.Info("Не удалось получить цену для %d товаров", pricesBad)
	logger.Info("Список этих товаров: %s", utilities.ToJSON(badLink))

	chLog <- fmt.Sprintf("Статистика по товарам успешно загружена...")
	chLog <- fmt.Sprintf("Начинаем генерацию таблицы...")
	logger.Info("Полученные каталоги: %s", utilities.ToJSON(catalogs))
	go generator.GenerateFile(catalogs, chLog)
}

func LoadCatalogs() ([]*types.Catalog, error) {
	doc, err := htmlquery.LoadURL(types.CATALOG_URL)
	if err != nil {
		return nil, err
	}

	var catalogs []*types.Catalog

	list := htmlquery.Find(doc, "//*[@class='dark_link']")
	for _, n := range list {
		catalog := new(types.Catalog)
		catalog.Name = htmlquery.InnerText(n)
		catalog.URL = fmt.Sprintf("%s%s", types.URL, n.Attr[0].Val)
		catalogs = append(catalogs, catalog)
	}

	return catalogs, nil
}

func LoadProduct(index int, url string, ch chan map[int]*types.Product, wg *sync.WaitGroup,
	chErr chan error) error {
	defer wg.Done()
	doc, err := htmlquery.LoadURL(url)
	if err != nil {
		return err
	}

	list := htmlquery.Find(doc, "//a[@class='dark_link js-notice-block__title option-font-bold font_sm']")
	if len(list) == 0 {
		logger.Warn("Не нашли товаров")
	}
	for _, n := range list {
		product := new(types.Product)
		product.Name = strings.Replace(htmlquery.InnerText(n), " ", " ", -1)
		product.URL = fmt.Sprintf("%s%s", types.URL, n.Attr[0].Val)
		//LoadPrice(product, nil, nil)

		mp := make(map[int]*types.Product, 1)
		mp[index] = product
		ch <- mp
	}

	list = nil

	//Обработка страниц
	{
		list = htmlquery.Find(doc, "//div[@class='module-pagination']")
		if len(list) == 0 {
			return nil
		} else {
			var urls []string
			list = htmlquery.Find(doc, "//div[@class='nums']/a/@href")
			for _, n := range list {
				urlCurrent := fmt.Sprintf("%s%s", types.URL, htmlquery.InnerText(n))
				urls = append(urls, urlCurrent)
			}

			for _, urlCurrent := range urls {
				doc, err = htmlquery.LoadURL(urlCurrent)
				if err != nil {
					logger.Warn("Ошибка при загрузке адреса: %s", urlCurrent)
					continue
				}

				list = htmlquery.Find(doc, "//a[@class='dark_link js-notice-block__title option-font-bold font_sm']")
				if len(list) == 0 {
					logger.Warn("Не нашли товаров")
					continue
				}

				for _, n := range list {
					product := new(types.Product)
					product.Name = strings.Replace(htmlquery.InnerText(n), " ", " ", -1)
					product.URL = fmt.Sprintf("%s%s", types.URL, n.Attr[0].Val)
					mp := make(map[int]*types.Product, 1)
					mp[index] = product
					ch <- mp
				}
			}
		}
	}

	return nil
}

func LoadPrice(product *types.Product, ch chan map[*types.Product]string, wg *sync.WaitGroup, chErr chan error) error {
	defer wg.Done()
	doc, err := htmlquery.LoadURL(product.URL)
	if err != nil {
		return err
	}

	arenda := false

	list := htmlquery.Find(doc, "//div[@class='price_matrix_block']")
	if len(list) == 0 {
		logger.Info("Для товара %s не нашли цену опт, ищем одну цену", product.URL)
		list = htmlquery.Find(doc, "//*[@id=\"content\"]/div[2]/div/div/div/div/div/div/div/div[2]/div/div[1]/div/div/div/div/span[1]") //"//div[@class='price_value_block values_wrapper']/span[@class='price_value']")
		if len(list) == 0 {
			list = htmlquery.Find(doc, "/html/body/div[5]/div[7]/div[2]/div/div/div/div/div/div/div/div[2]/div/div[1]/div/div/div/div/span[1]")
			if len(list) == 0 {
				list = htmlquery.Find(doc, "/html/body/div[5]/div[6]/div[2]/div/div/div/div/div/div/div/div[2]/div/div[1]/div/div/div/div/span[1]")
			}
		}

		if len(list) == 0 {
			list = htmlquery.Find(doc, "//*[@class='srok-price-initial']")
			if len(list) != 0 {
				arenda = true
			}
			if len(list) == 0 {
				list = htmlquery.Find(doc,
					"/html/body/div[5]/div[7]/div[2]/div/div/div/div/div/div/div/div[2]/div/div[2]/div[3]")
				if len(list) != 0 {
					arenda = true
				}
			}
		}

		if len(list) == 0 {
			list = htmlquery.Find(doc, "//*[@class='price_value_block values_wrapper']")
		}

		zakaz := false

		if len(list) == 0 {
			list = htmlquery.Find(doc, "//span[@class='store_view dotted']")
			if len(list) != 0 {
				zakaz = true
			}
		}

		if len(list) == 0 {
			logger.Warn("Цену не нашли: %s", product.URL)
		}

		for _, n := range list {
			var price string
			if !arenda && !zakaz {
				price = fmt.Sprintf("%s ₽", htmlquery.InnerText(n))
			}
			price = htmlquery.InnerText(n)
			price = strings.Replace(price, " ", " ", -1)
			price = strings.Replace(price,
				"\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t", "", -1)
			mp := make(map[*types.Product]string, 1)
			mp[product] = price
			ch <- mp
		}
	} else {
		logger.Info("Для товара %s нашли цену", product.URL)
		for _, n := range list {
			text := htmlquery.InnerText(n)
			format1 := strings.Replace(text,
				"\n\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t"+
					"\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t", "шт - ", -1)
			format2 := strings.Replace(format1,
				"\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t", "", -1)
			format3 := strings.Replace(format2,
				"\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t"+
					"\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t"+
					"\t\t\t\t\t\t\t\t\t\t\t\t", "\n", -1)
			format4 := strings.Replace(format3,
				"\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t", "", -1)
			format5 := strings.Replace(format4,
				"\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t", "", -1)

			mp := make(map[*types.Product]string, 1)
			mp[product] = format5
			ch <- mp
		}
	}

	return nil
}
