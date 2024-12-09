package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GeocodingService struct{}

func (s *GeocodingService) GetAddressFromLatLng(lat, lng float64) (map[string]string, error) {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	url := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&key=%s&language=ja", lat, lng, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("住所の取得に失敗しました")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("住所の取得に失敗しました")
	}

	var result struct {
		Results []struct {
			AddressComponents []struct {
				LongName string   `json:"long_name"`
				Types    []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("住所データの解析に失敗しました")
	}

	var (
		prefecture string
		city       string
		town       string
		chome      string
		banchi     string
		postalCode string
	)

	for _, component := range result.Results[0].AddressComponents {
		types := component.Types

		if contains(types, "administrative_area_level_1") {
			prefecture = component.LongName
		} else if contains(types, "locality") {
			city = component.LongName
		} else if contains(types, "sublocality_level_2") {
			town = component.LongName
		} else if contains(types, "sublocality_level_3") {
			chome = component.LongName
		} else if contains(types, "premise") {
			banchi = component.LongName
		} else if contains(types, "street_number") {
			banchi = component.LongName
		} else if contains(types, "postal_code") {
			postalCode = component.LongName
		}
	}

	address := fmt.Sprintf("%s%s%s%s%s", prefecture, city, town, chome, banchi)

	return map[string]string{
		"address":    address,
		"postalCode": postalCode,
	}, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
