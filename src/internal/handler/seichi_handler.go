// Seicheese-Backend/src/internal/handler/seichi_handler.go
package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"seicheese/models"
	"strconv"
	"strings"
	"time"

	"github.com/ericlagergren/decimal"
	"github.com/labstack/echo/v4"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/types"
)

type SeichiResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	ContentID   int       `json:"content_id"`
	ContentName string    `json:"content_name"`
	Address     string    `json:"address"`
	PostalCode  string    `json:"postal_code"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SeichiHandler struct {
	DB *sql.DB
}

// 既存の場所を検索する関数を追加
func (h *SeichiHandler) findExistingPlace(ctx context.Context, address string) (*models.Place, error) {
	return models.Places(
		models.PlaceWhere.Address.EQ(address),
	).One(ctx, h.DB)
}

// 住所文字列から都道府県から番地までを抽出する関数を追加
func extractAddress(fullAddress string) string {
	// "日本、〒000-0000 "のような部分を削除
	if idx := strings.Index(fullAddress, "〒"); idx != -1 {
		if spaceIdx := strings.Index(fullAddress[idx:], " "); spaceIdx != -1 {
			fullAddress = fullAddress[idx+spaceIdx+1:]
		}
	}

	// "日本、"を削除
	fullAddress = strings.TrimPrefix(fullAddress, "日本、")

	return fullAddress
}

type RegisterSeichiRequest struct {
	Name        string  `json:"seichi_name"`
	Description string  `json:"comment"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ContentID   int     `json:"content_id"`
}

// 聖地登録API
func (h *SeichiHandler) RegisterSeichi(c echo.Context) error {
	// リクエストのデバッグ出力
	body, _ := io.ReadAll(c.Request().Body)
	log.Printf("Raw request body: %s", string(body))
	c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

	var req RegisterSeichiRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// 値の確認
	log.Printf("Bound request data: %+v", req)

	log.Printf("Received seichi name: %s", req.Name)
	log.Printf("Received seichi description: %s", req.Description)
	log.Printf("Received seichi latitude: %f", req.Latitude)
	log.Printf("Received seichi longitude: %f", req.Longitude)
	log.Printf("Received seichi contentID: %d", req.ContentID)

	// Firebase UIDからユーザーIDを取得
	uid := c.Get("uid").(string)
	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(c.Request().Context(), h.DB)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "ユーザー情報の取得に失敗しました",
		})
	}

	log.Printf("Fetched user: %v", user)

	// 住所情報を取得
	addressData, err := getAddressFromCoordinates(req.Latitude, req.Longitude)
	if err != nil {
		log.Printf("Error getting address: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "住所の取得に失敗しました",
		})
	}

	log.Printf("Fetched address data: %v", addressData)

	// 住所データ取得後の処理
	var place *models.Place
	existingPlace, err := h.findExistingPlace(c.Request().Context(), extractAddress(addressData["address"]))
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error querying existing place: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "場所の検索に失敗しました",
			})
		}
		// 新しい場所を登録
		place = &models.Place{
			Address: extractAddress(addressData["address"]),
			ZipCode: addressData["postalCode"],
		}
		if err := place.Insert(c.Request().Context(), h.DB, boil.Infer()); err != nil {
			log.Printf("Error inserting place: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "場所の登録に失敗しました",
			})
		}
		log.Printf("Inserted new place: %v", place)
	} else {
		place = existingPlace
		log.Printf("Using existing place: %v", place)
	}

	// 日本周辺の緯度経度の範囲を確認
	if req.Latitude < 24.396308 || req.Latitude > 45.551483 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "緯度の値が日本の範囲外です",
		})
	}
	if req.Longitude < 122.93457 || req.Longitude > 153.986672 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "経度の値が日本の範囲外です",
		})
	}

	latitudeDecimal := new(decimal.Big).SetFloat64(req.Latitude)
	longitudeDecimal := new(decimal.Big).SetFloat64(req.Longitude)

	log.Printf("Latitude decimal: %s", latitudeDecimal.String())
	log.Printf("Longitude decimal: %s", longitudeDecimal.String())

	// 聖地を登録
	seichi := &models.Seichy{
		UserID:     user.UserID,
		SeichiName: req.Name,
		Comment:    null.StringFrom(req.Description),
		Latitude:   types.Decimal{Big: latitudeDecimal},
		Longitude:  types.Decimal{Big: longitudeDecimal},
		PlaceID:    place.PlaceID,
		ContentID:  req.ContentID,
	}

	if err := seichi.Insert(c.Request().Context(), h.DB, boil.Infer()); err != nil {
		log.Printf("Failed to create seichi: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to register seichi")
	}

	log.Printf("Seichi registered successfully: %v", seichi)

	var comment string
	if seichi.Comment.Valid {
		comment = seichi.Comment.String
	} else {
		comment = ""
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"seichi_id":   seichi.SeichiID,
		"user_id":     seichi.UserID,
		"seichi_name": seichi.SeichiName,
		"comment":     comment,
		"latitude":    seichi.Latitude,
		"longitude":   seichi.Longitude,
		"place_id":    seichi.PlaceID,
		"content_id":  seichi.ContentID,
		"created_at":  seichi.CreatedAt,
		"updated_at":  seichi.UpdatedAt,
	})
}

// 聖地取得API
func (h *SeichiHandler) GetSeichies(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit == 0 {
		limit = 50 // デフォルトの取得件数
	}
	if page == 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// 表示範囲内のデータのみを取得
	seichies, err := models.Seichies(
		qm.Load("Content"),
		qm.Load("Place"),
		qm.Limit(limit),
		qm.Offset(offset),
	).All(c.Request().Context(), h.DB)

	if err != nil {
		log.Printf("Error fetching seichies: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "聖地データの取得に失敗しました",
		})
	}

	var response []map[string]interface{}
	for _, s := range seichies {
		contentName := ""
		if s.R != nil && s.R.Content != nil {
			contentName = s.R.Content.ContentName
		}

		address := ""
		postalCode := ""
		if s.R != nil && s.R.Place != nil {
			address = s.R.Place.Address
			postalCode = s.R.Place.ZipCode
		}

		latitude, _ := s.Latitude.Float64()
		longitude, _ := s.Longitude.Float64()

		response = append(response, map[string]interface{}{
			"id":           s.SeichiID,
			"name":         s.SeichiName,
			"description":  s.Comment.String,
			"latitude":     latitude,
			"longitude":    longitude,
			"content_id":   s.ContentID,
			"content_name": contentName,
			"address":      address,
			"postal_code":  postalCode,
			"created_at":   s.CreatedAt.Time.Format(time.RFC3339),
			"updated_at":   s.UpdatedAt.Time.Format(time.RFC3339),
		})
	}

	c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	return c.JSON(http.StatusOK, response)
}

// クラスタリング処理を行う関数
func (h *SeichiHandler) getClusteredSeichies(ctx context.Context, bounds string) ([]map[string]interface{}, error) {
	// SQLでグリッドベースのクラスタリングを実行
	query := `
	SELECT 
		ROUND(AVG(CAST(latitude AS FLOAT)), 4) as lat,
		ROUND(AVG(CAST(longitude AS FLOAT)), 4) as lng,
		COUNT(*) as count
	FROM seichies
	GROUP BY 
		FLOOR(CAST(latitude AS FLOAT) * 100),
		FLOOR(CAST(longitude AS FLOAT) * 100)
	HAVING count > 1
	`

	rows, err := h.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []map[string]interface{}
	for rows.Next() {
		var lat, lng float64
		var count int
		if err := rows.Scan(&lat, &lng, &count); err != nil {
			return nil, err
		}
		clusters = append(clusters, map[string]interface{}{
			"latitude":   lat,
			"longitude":  lng,
			"count":      count,
			"is_cluster": true,
		})
	}
	return clusters, nil
}

func getAddressFromCoordinates(lat, lng float64) (map[string]string, error) {
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s&language=ja&result_type=postal_code|administrative_area_level_1|locality|sublocality|street_number",
		lat, lng, os.Getenv("GOOGLE_MAPS_API_KEY"),
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			AddressComponents []struct {
				LongName string   `json:"long_name"`
				Types    []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no results found")
	}

	var (
		prefecture string
		city      string
		district  string
		street    string
		number    string
		postal    string
	)

	// 住所コンポーネントを解析
	for _, component := range result.Results[0].AddressComponents {
		for _, type_ := range component.Types {
			switch type_ {
			case "postal_code":
				postal = component.LongName
			case "administrative_area_level_1":
				prefecture = component.LongName
			case "locality":
				city = component.LongName
			case "sublocality_level_1":
				district = component.LongName
			case "sublocality_level_2":
				if street == "" {
					street = component.LongName
				}
			case "street_number":
				number = component.LongName
			}
		}
	}

	// 住所を組み立て
	var addressParts []string
	if prefecture != "" {
		addressParts = append(addressParts, prefecture)
	}
	if city != "" {
		addressParts = append(addressParts, city)
	}
	if district != "" {
		addressParts = append(addressParts, district)
	}
	if street != "" {
		addressParts = append(addressParts, street)
	}
	if number != "" {
		addressParts = append(addressParts, number)
	}

	address := strings.Join(addressParts, "")

	return map[string]string{
		"address":    address,
		"postalCode": postal,
	}, nil
}

// SearchSeichis handles the search endpoint
func (h *SeichiHandler) SearchSeichis(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "検索クエリが必要です"})
	}

	ctx := c.Request().Context()

	// SQLBoilerのクエリビルダーを使用
	seichis, err := models.Seichies(
		qm.Load("Content"),
		qm.Load("Content.Genre"),
		qm.Where(
			"seichi_name LIKE ? OR "+
				"address LIKE ? OR "+
				"postal_code LIKE ? OR "+
				"EXISTS (SELECT 1 FROM contents WHERE contents.id = seichis.content_id AND contents.name LIKE ?) OR "+
				"EXISTS (SELECT 1 FROM contents c JOIN genres g ON c.genre_id = g.id WHERE c.id = seichis.content_id AND g.name LIKE ?)",
			"%"+query+"%",
			"%"+query+"%",
			"%"+query+"%",
			"%"+query+"%",
			"%"+query+"%",
		),
		qm.OrderBy("seichis.created_at DESC"),
	).All(ctx, h.DB)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "検索中にエラーが発生しました"})
	}

	return c.JSON(http.StatusOK, seichis)
}
