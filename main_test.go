package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockRoundTripper struct{}

func (mrt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Host {
	case "viacep.com.br":
		body := `{"localidade": "Sao Paulo"}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	case "api.weatherapi.com":
		body := `{"current": {"temp_c": 25}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	default:
		return nil, fmt.Errorf("unexpected host: %s", req.URL.Host)
	}
}

func TestInvalidZipcode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/weather", weatherHandler)

	req := httptest.NewRequest("GET", "/weather?cep=123", nil)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
	var res map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if res["message"] != "invalid zipcode" {
		t.Errorf("expected message 'invalid zipcode', got '%s'", res["message"])
	}
}

func TestValidWeather(t *testing.T) {
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()
	http.DefaultTransport = &mockRoundTripper{}
	os.Setenv("WEATHER_API_KEY", "dummy")

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/weather", weatherHandler)

	req := httptest.NewRequest("GET", "/weather?cep=12345678", nil)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = req

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	var res WeatherResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	expectedC := 25.0
	expectedF := expectedC*1.8 + 32
	expectedK := expectedC + 273
	if res.TempC != expectedC || res.TempF != expectedF || res.TempK != expectedK {
		t.Errorf("expected temperatures {TempC: %.2f, TempF: %.2f, TempK: %.2f}, got {TempC: %.2f, TempF: %.2f, TempK: %.2f}",
			expectedC, expectedF, expectedK, res.TempC, res.TempF, res.TempK)
	}
}
