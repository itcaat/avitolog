package parser

import (
	"net/url"
	"strings"
	
	"github.com/itcaat/avitolog/internal/models"
)

const (
	baseURL = "https://www.avito.ru"
)

// GetCategories returns a predefined list of main categories and their subcategories from Avito.ru
func GetCategories() ([]models.Category, error) {
	// Define the main categories with their common subcategories
	// This structure is based on the actual categories visible on Avito.ru
	return []models.Category{
		{
			Name: "Транспорт",
			URL:  "https://www.avito.ru/all/transport",
			Subcategories: []models.Category{
				{Name: "Автомобили", URL: "https://www.avito.ru/all/avtomobili"},
				{Name: "Мотоциклы и мототехника", URL: "https://www.avito.ru/all/mototsikly_i_mototehnika"},
				{Name: "Грузовики и спецтехника", URL: "https://www.avito.ru/all/gruzoviki_i_spetstehnika"},
				{Name: "Водный транспорт", URL: "https://www.avito.ru/all/vodnyy_transport"},
				{Name: "Запчасти и аксессуары", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary"},
			},
		},
		{
			Name: "Недвижимость",
			URL:  "https://www.avito.ru/all/nedvizhimost",
			Subcategories: []models.Category{
				{Name: "Купить жильё", URL: "https://www.avito.ru/all/nedvizhimost/kvartiry/prodam"},
				{Name: "Путешествия", URL: "https://www.avito.ru/all/nedvizhimost/kvartiry/posutochno"},
				{Name: "Снять долгосрочно", URL: "https://www.avito.ru/all/nedvizhimost/kvartiry/sdam"},
				{Name: "Коммерческая недвижимость", URL: "https://www.avito.ru/all/kommercheskaya_nedvizhimost"},
				{Name: "Дома, дачи, коттеджи", URL: "https://www.avito.ru/all/doma_dachi_kottedzhi"},
				{Name: "Земельные участки", URL: "https://www.avito.ru/all/zemelnye_uchastki"},
				{Name: "Гаражи и машиноместа", URL: "https://www.avito.ru/all/garazhi_i_mashinomesta"},
			},
		},
		{
			Name: "Работа",
			URL:  "https://www.avito.ru/all/rabota",
			Subcategories: []models.Category{
				{Name: "Вакансии", URL: "https://www.avito.ru/all/vakansii"},
				{Name: "Резюме", URL: "https://www.avito.ru/all/rezume"},
			},
		},
		{
			Name: "Услуги",
			URL:  "https://www.avito.ru/all/predlozheniya_uslug",
			Subcategories: []models.Category{
				{Name: "Строительство и ремонт", URL: "https://www.avito.ru/all/predlozheniya_uslug/stroitelstvo_remont_montazh"},
				{Name: "Перевозки и аренда транспорта", URL: "https://www.avito.ru/all/predlozheniya_uslug/perevozki_i_arenda_transporta"},
				{Name: "Красота и здоровье", URL: "https://www.avito.ru/all/predlozheniya_uslug/krasota_zdorove"},
				{Name: "Обучение и курсы", URL: "https://www.avito.ru/all/predlozheniya_uslug/obuchenie_kursy"},
				{Name: "Установка техники", URL: "https://www.avito.ru/all/predlozheniya_uslug/remont_i_obsluzhivanie_tehniki"},
				{Name: "Деловые услуги", URL: "https://www.avito.ru/all/predlozheniya_uslug/delovye_uslugi"},
				{Name: "Мастер на час", URL: "https://www.avito.ru/all/predlozheniya_uslug/master_na_chas"},
				{Name: "Автосервис", URL: "https://www.avito.ru/all/predlozheniya_uslug/transport_perevozki/avtoservis"},
				{Name: "Уборка и помощь по хозяйству", URL: "https://www.avito.ru/all/predlozheniya_uslug/bytovye_uslugi"},
			},
		},
		{
			Name: "Личные вещи",
			URL:  "https://www.avito.ru/all/lichnye_veschi",
			Subcategories: []models.Category{
				{Name: "Одежда, обувь, аксессуары", URL: "https://www.avito.ru/all/odezhda_obuv_aksessuary"},
				{Name: "Детская одежда и обувь", URL: "https://www.avito.ru/all/detskaya_odezhda_i_obuv"},
				{Name: "Товары для детей и игрушки", URL: "https://www.avito.ru/all/tovary_dlya_detey_i_igrushki"},
				{Name: "Часы и украшения", URL: "https://www.avito.ru/all/chasy_i_ukrasheniya"},
				{Name: "Красота и здоровье", URL: "https://www.avito.ru/all/krasota_i_zdorove"},
			},
		},
		{
			Name: "Для дома и дачи",
			URL:  "https://www.avito.ru/all/dlya_doma_i_dachi",
			Subcategories: []models.Category{
				{Name: "Бытовая техника", URL: "https://www.avito.ru/all/bytovaya_tehnika"},
				{Name: "Мебель и интерьер", URL: "https://www.avito.ru/all/mebel_i_interer"},
				{Name: "Посуда и товары для кухни", URL: "https://www.avito.ru/all/posuda_i_tovary_dlya_kuhni"},
				{Name: "Продукты питания", URL: "https://www.avito.ru/all/produkty_pitaniya"},
				{Name: "Ремонт и строительство", URL: "https://www.avito.ru/all/remont_i_stroitelstvo"},
				{Name: "Растения", URL: "https://www.avito.ru/all/rasteniya"},
			},
		},
		{
			Name: "Запчасти и аксессуары",
			URL:  "https://www.avito.ru/all/zapchasti_i_aksessuary",
			Subcategories: []models.Category{
				{Name: "Запчасти", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary/zapchasti"},
				{Name: "Шины, диски и колёса", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary/shiny_diski_i_kolesa"},
				{Name: "Аксессуары", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary/aksessuary"},
				{Name: "Автозвук", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary/avtozvuk"},
				{Name: "Инструменты", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary/instrumenty"},
				{Name: "Мототехника", URL: "https://www.avito.ru/all/zapchasti_i_aksessuary/mototsikly_mopedy/zapchasti_i_aksessuary"},
			},
		},
		{
			Name: "Электроника",
			URL:  "https://www.avito.ru/all/bytovaya_elektronika",
			Subcategories: []models.Category{
				{Name: "Аудио и видео", URL: "https://www.avito.ru/all/audio_i_video"},
				{Name: "Игры, приставки и программы", URL: "https://www.avito.ru/all/igry_pristavki_i_programmy"},
				{Name: "Компьютеры, ноутбуки и ПО", URL: "https://www.avito.ru/all/nastolnye_kompyutery"},
				{Name: "Оргтехника и расходники", URL: "https://www.avito.ru/all/orgtehnika_i_rashodniki"},
				{Name: "Планшеты и электронные книги", URL: "https://www.avito.ru/all/planshety_i_elektronnye_knigi"},
				{Name: "Телефоны", URL: "https://www.avito.ru/all/telefony"},
				{Name: "Товары для компьютера", URL: "https://www.avito.ru/all/tovary_dlya_kompyutera"},
				{Name: "Фототехника", URL: "https://www.avito.ru/all/fototehnika"},
			},
		},
		{
			Name: "Хобби и отдых",
			URL:  "https://www.avito.ru/all/hobbi_i_otdyh",
			Subcategories: []models.Category{
				{Name: "Билеты и путешествия", URL: "https://www.avito.ru/all/bilety_i_puteshestviya"},
				{Name: "Велосипеды", URL: "https://www.avito.ru/all/velosipedy"},
				{Name: "Книги и журналы", URL: "https://www.avito.ru/all/knigi_i_zhurnaly"},
				{Name: "Коллекционирование", URL: "https://www.avito.ru/all/kollektsionirovanie"},
				{Name: "Музыкальные инструменты", URL: "https://www.avito.ru/all/muzykalnye_instrumenty"},
				{Name: "Охота и рыбалка", URL: "https://www.avito.ru/all/ohota_i_rybalka"},
				{Name: "Спорт и отдых", URL: "https://www.avito.ru/all/sport_i_otdyh"},
			},
		},
		{
			Name: "Животные",
			URL:  "https://www.avito.ru/all/zhivotnye",
			Subcategories: []models.Category{
				{Name: "Собаки", URL: "https://www.avito.ru/all/sobaki"},
				{Name: "Кошки", URL: "https://www.avito.ru/all/koshki"},
				{Name: "Аквариумные рыбки", URL: "https://www.avito.ru/all/akvariumnye_rybki"},
				{Name: "Птицы", URL: "https://www.avito.ru/all/ptitsy"},
				{Name: "Грызуны", URL: "https://www.avito.ru/all/gryzuny"},
				{Name: "Товары для животных", URL: "https://www.avito.ru/all/tovary_dlya_zhivotnyh"},
			},
		},
		{
			Name: "Для бизнеса",
			URL:  "https://www.avito.ru/all/dlya_biznesa",
			Subcategories: []models.Category{
				{Name: "Готовый бизнес", URL: "https://www.avito.ru/all/gotoviy_biznes"},
				{Name: "Оборудование для бизнеса", URL: "https://www.avito.ru/all/oborudovanie_dlya_biznesa"},
			},
		},
	}, nil
}

// normalizeURL ensures the URL is absolute
func normalizeURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}
	
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	
	if strings.HasPrefix(href, "/") {
		return baseURL + href
	}
	
	// Try to parse the URL to handle other cases
	parsedURL, err := url.Parse(href)
	if err != nil {
		return baseURL + "/" + href
	}
	
	// If parsed successfully but is relative
	if !parsedURL.IsAbs() {
		return baseURL + "/" + href
	}
	
	return href
}
