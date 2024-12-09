package handler

import (
	"database/sql"
	"log"
	"net/http"
	"seicheese/models"
	"seicheese/services"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type PlaceHandler struct {
	DB *sql.DB
}

// Place登録API
func (h *PlaceHandler) RegisterPlace(c echo.Context) error {
	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}

	if err := c.Bind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "不正なリクエスト形式です",
		})
	}

	// 緯度経度を使用して住所と郵便番号を検索
	geocodingService := services.GeocodingService{}
	addressData, err := geocodingService.GetAddressFromLatLng(req.Latitude, req.Longitude)
	if err != nil {
		log.Printf("Error fetching address: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "住所の取得に失敗しました",
		})
	}

	// 住所フォーマットを整形する関数
	formatAddress := func(components map[string]string) string {
		var address strings.Builder

		// 都道府県
		if prefecture, ok := components["administrative_area_level_1"]; ok {
			address.WriteString(prefecture)
		}

		// 市区町村
		if city, ok := components["locality"]; ok {
			address.WriteString(city)
		}

		// 町名
		if sublocality, ok := components["sublocality_level_1"]; ok {
			address.WriteString(sublocality)
		}

		// 番地
		if streetNumber, ok := components["street_number"]; ok {
			address.WriteString(streetNumber)
		}

		return address.String()
	}

	// Google Maps APIからの応答を処理
	formattedAddress := formatAddress(addressData)

	place := &models.Place{
		Address: formattedAddress,
		ZipCode: addressData["postalCode"],
	}

	if err := place.Insert(c.Request().Context(), h.DB, boil.Infer()); err != nil {
		log.Printf("Error inserting place: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Placeの登録に失敗しました",
		})
	}

	log.Printf("Place registered successfully: %v", place)
	return c.JSON(http.StatusCreated, place)
}

func (h *PlaceHandler) GetPlace(c echo.Context) error {
	places, err := models.Places().All(c.Request().Context(), h.DB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "場所の取得に失敗しました")
	}

	return c.JSON(http.StatusOK, places)
}
