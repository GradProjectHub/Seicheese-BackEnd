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
	GenreID     int       `json:"genre_id"`
	GenreName   string    `json:"genre_name"`
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
	log.Printf("Attempting to fetch user with Firebase UID: %s", uid)

	user, err := models.Users(
		models.UserWhere.FirebaseID.EQ(uid),
	).One(c.Request().Context(), h.DB)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No user found with Firebase UID: %s, creating new user", uid)
			// ユーザーが存在しない場合は新規登録を試みる
			now := time.Now()
			newUser := &models.User{
				FirebaseID: uid,
				CreatedAt:  null.TimeFrom(now),
				UpdatedAt:  null.TimeFrom(now),
			}

			if err := newUser.Insert(c.Request().Context(), h.DB, boil.Infer()); err != nil {
				log.Printf("Failed to create new user: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "ユーザーの登録に失敗しました",
				})
			}
			log.Printf("Created new user: %+v", newUser)
			user = newUser
		} else {
			log.Printf("Error fetching user: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "ユーザー情報の取得に失敗しました",
			})
		}
	} else {
		log.Printf("Found existing user: %+v", user)
	}

	log.Printf("Using user: %+v", user)

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
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s&language=ja&region=jp",
		lat, lng, apiKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Geocoding API request failed: %v", err)
		return nil, fmt.Errorf("住所情報の取得に失敗しました: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return nil, fmt.Errorf("レスポンスの読み取りに失敗しました: %v", err)
	}
	log.Printf("Geocoding API response: %s", string(body))

	var result struct {
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
			AddressComponents []struct {
				LongName string   `json:"long_name"`
				Types    []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		log.Printf("Failed to decode response: %v", err)
		return nil, fmt.Errorf("レスポンスのデコードに失敗しました: %v", err)
	}

	if result.Status != "OK" {
		log.Printf("Geocoding API returned non-OK status: %s", result.Status)
		return nil, fmt.Errorf("住所情報の取得に失敗しました（ステータス: %s）", result.Status)
	}

	if len(result.Results) == 0 {
		log.Printf("No results found in Geocoding API response")
		return nil, fmt.Errorf("指定された座標の住所情報が見つかりませんでした")
	}

	var (
		prefecture string
		city       string
		district   string
		street     string
		chome      string
		block      string
		number     string
		postal     string
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
				street = component.LongName
			case "sublocality_level_3":
				chome = component.LongName
			case "sublocality_level_4":
				block = component.LongName
			case "premise":
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
	if chome != "" {
		addressParts = append(addressParts, strings.ReplaceAll(chome, "丁目", "丁目"))
	}
	if block != "" && number != "" {
		addressParts = append(addressParts, fmt.Sprintf("%s-%s", block, number))
	} else {
		if block != "" {
			addressParts = append(addressParts, fmt.Sprintf("%s番", block))
		}
		if number != "" {
			addressParts = append(addressParts, fmt.Sprintf("%s号", number))
		}
	}

	// デバッグログを追加
	log.Printf("Address components: prefecture=%s, city=%s, district=%s, street=%s, chome=%s, block=%s, number=%s",
		prefecture, city, district, street, chome, block, number)

	address := strings.Join(addressParts, "")

	return map[string]string{
		"address":    address,
		"postalCode": postal,
	}, nil
}

// SearchSeichies は聖地を検索するハンドラー
func (h *SeichiHandler) SearchSeichies(c echo.Context) error {
	// コンテキストにタイムアウトを設定
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// クエリパラメータの取得
	query := c.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "検索キーワードが必要です")
	}

	// SQLインジェクション対策のためにワイルドカードをエスケープ
	query = strings.ReplaceAll(query, "%", "\\%")
	query = strings.ReplaceAll(query, "_", "\\_")
	searchPattern := "%" + query + "%"

	// Eager Loadingを使用して関連データを事前に読み込む
	seichies, err := models.Seichies(
		qm.Load("Content"),
		qm.Load("Content.Genre"),
		qm.Load("Place"),
		qm.Select(
			"DISTINCT seichies.*",
			"contents.content_name",
			"genres.genre_name",
			"places.address",
			"places.zip_code",
		),
		qm.InnerJoin("contents on seichies.content_id = contents.content_id"),
		qm.InnerJoin("genres on contents.genre_id = genres.genre_id"),
		qm.InnerJoin("places on seichies.place_id = places.place_id"),
		qm.Where(
			"(seichies.seichi_name LIKE ? OR contents.content_name LIKE ? OR places.address LIKE ?) AND LENGTH(?) >= 2",
			searchPattern, searchPattern, searchPattern, query,
		),
		qm.OrderBy("seichies.created_at DESC"),
		qm.Limit(20),
	).All(ctx, h.DB)

	if err != nil {
		log.Printf("Failed to search seichies: %v, query: %s", err, query)
		return echo.NewHTTPError(http.StatusInternalServerError, "聖地の検索に失敗しました")
	}

	response := make([]SeichiResponse, 0, len(seichies))
	for _, s := range seichies {
		if s.R == nil || s.R.Content == nil || s.R.Place == nil || s.R.Content.R.Genre == nil {
			log.Printf("Warning: Skipping seichi %d due to missing related data", s.SeichiID)
			continue
		}

		latitude, _ := s.Latitude.Float64()
		longitude, _ := s.Longitude.Float64()

		response = append(response, SeichiResponse{
			ID:          s.SeichiID,
			Name:        s.SeichiName,
			Description: s.Comment.String,
			Latitude:    latitude,
			Longitude:   longitude,
			ContentID:   s.ContentID,
			ContentName: s.R.Content.ContentName,
			GenreID:     s.R.Content.GenreID,
			GenreName:   s.R.Content.R.Genre.GenreName,
			Address:     s.R.Place.Address,
			PostalCode:  s.R.Place.ZipCode,
			CreatedAt:   s.CreatedAt.Time,
			UpdatedAt:   s.UpdatedAt.Time,
		})
	}

	return c.JSON(http.StatusOK, response)
}
