package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Keneke-Einar/delivertrack/pkg/logger"
	"go.uber.org/zap"
)

// GeocodingService interface for address ↔ coordinate conversion
type GeocodingService interface {
	ForwardGeocode(ctx context.Context, address string) (*GeocodeResult, error)
	ReverseGeocode(ctx context.Context, lat, lng float64) (*ReverseGeocodeResult, error)
	Autocomplete(ctx context.Context, query string) ([]AutocompleteResult, error)
}

// GeocodeResult represents a geocoding response
type GeocodeResult struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Country   string  `json:"country"`
	ZipCode   string  `json:"zip_code"`
}

// ReverseGeocodeResult represents reverse geocoding response
type ReverseGeocodeResult struct {
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code"`
}

// AutocompleteResult for address suggestions
type AutocompleteResult struct {
	Text        string `json:"text"`
	Description string `json:"description"`
}

// NominatimResponse represents the response from Nominatim API
type NominatimResponse struct {
	PlaceID     int              `json:"place_id"`
	License     string           `json:"licence"`
	OSMType     string           `json:"osm_type"`
	OSMID       int              `json:"osm_id"`
	Lat         string           `json:"lat"`
	Lon         string           `json:"lon"`
	Class       string           `json:"class"`
	Type        string           `json:"type"`
	PlaceRank   int              `json:"place_rank"`
	Importance  float64          `json:"importance"`
	Addresstype string           `json:"addresstype"`
	Name        string           `json:"name"`
	DisplayName string           `json:"display_name"`
	Address     NominatimAddress `json:"address"`
	Boundingbox []string         `json:"boundingbox"`
}

// NominatimAddress represents the address part of Nominatim response
type NominatimAddress struct {
	HouseNumber string `json:"house_number"`
	Road        string `json:"road"`
	Suburb      string `json:"suburb"`
	City        string `json:"city"`
	State       string `json:"state"`
	Postcode    string `json:"postcode"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

// HTTPGeocodingService implements GeocodingService using external APIs
type HTTPGeocodingService struct {
	client *http.Client
	logger *logger.Logger
}

// NewHTTPGeocodingService creates a geocoding service using free APIs
func NewHTTPGeocodingService(logger *logger.Logger) *HTTPGeocodingService {
	return &HTTPGeocodingService{
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

// ForwardGeocode converts address to coordinates using Nominatim (OpenStreetMap)
func (s *HTTPGeocodingService) ForwardGeocode(ctx context.Context, address string) (*GeocodeResult, error) {
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Build Nominatim API URL
	baseURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Add("format", "json")
	params.Add("q", address)
	params.Add("limit", "1")
	params.Add("addressdetails", "1")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	s.logger.InfoWithFields(ctx, "Geocoding address",
		zap.String("address", address), zap.String("url", fullURL))

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent as required by Nominatim
	req.Header.Set("User-Agent", "DeliverTrack/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call geocoding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding API returned status %d", resp.StatusCode)
	}

	var results []NominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode geocoding response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for address: %s", address)
	}

	result := results[0]

	// Parse latitude and longitude
	lat, err := parseFloat(result.Lat)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude in response: %s", result.Lat)
	}

	lng, err := parseFloat(result.Lon)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude in response: %s", result.Lon)
	}

	// Extract address components
	addr := result.Address
	geocodeResult := &GeocodeResult{
		Latitude:  lat,
		Longitude: lng,
		Address:   result.DisplayName,
		City:      addr.City,
		State:     addr.State,
		Country:   addr.Country,
		ZipCode:   addr.Postcode,
	}

	s.logger.InfoWithFields(ctx, "Successfully geocoded address",
		zap.String("address", address),
		zap.Float64("lat", lat),
		zap.Float64("lng", lng),
		zap.String("display_name", result.DisplayName))

	return geocodeResult, nil
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// ReverseGeocode converts coordinates to address using Nominatim
func (s *HTTPGeocodingService) ReverseGeocode(ctx context.Context, lat, lng float64) (*ReverseGeocodeResult, error) {
	// Build Nominatim reverse API URL
	baseURL := "https://nominatim.openstreetmap.org/reverse"
	params := url.Values{}
	params.Add("format", "json")
	params.Add("lat", fmt.Sprintf("%.6f", lat))
	params.Add("lon", fmt.Sprintf("%.6f", lng))
	params.Add("addressdetails", "1")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	s.logger.InfoWithFields(ctx, "Reverse geocoding coordinates",
		zap.Float64("lat", lat), zap.Float64("lng", lng), zap.String("url", fullURL))

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "DeliverTrack/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call reverse geocoding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reverse geocoding API returned status %d", resp.StatusCode)
	}

	var result NominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode reverse geocoding response: %w", err)
	}

	if result.DisplayName == "" {
		return nil, fmt.Errorf("no address found for coordinates: %f, %f", lat, lng)
	}

	// Extract address components
	addr := result.Address
	reverseResult := &ReverseGeocodeResult{
		Address: result.DisplayName,
		City:    addr.City,
		State:   addr.State,
		Country: addr.Country,
		ZipCode: addr.Postcode,
	}

	s.logger.InfoWithFields(ctx, "Successfully reverse geocoded coordinates",
		zap.Float64("lat", lat), zap.Float64("lng", lng), zap.String("address", result.DisplayName))

	return reverseResult, nil
}

// Autocomplete provides address suggestions using Nominatim search
func (s *HTTPGeocodingService) Autocomplete(ctx context.Context, query string) ([]AutocompleteResult, error) {
	if query == "" {
		return []AutocompleteResult{}, nil
	}

	// Build Nominatim search API URL for suggestions
	baseURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Add("format", "json")
	params.Add("q", query)
	params.Add("limit", "5") // Get up to 5 suggestions
	params.Add("addressdetails", "1")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	s.logger.InfoWithFields(ctx, "Getting address suggestions",
		zap.String("query", query), zap.String("url", fullURL))

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "DeliverTrack/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call autocomplete API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("autocomplete API returned status %d", resp.StatusCode)
	}

	var results []NominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode autocomplete response: %w", err)
	}

	var suggestions []AutocompleteResult
	for _, result := range results {
		suggestions = append(suggestions, AutocompleteResult{
			Text:        result.DisplayName,
			Description: fmt.Sprintf("%s (%s)", result.Addresstype, result.Type),
		})
	}

	s.logger.InfoWithFields(ctx, "Got address suggestions",
		zap.String("query", query), zap.Int("count", len(suggestions)))

	return suggestions, nil
}
