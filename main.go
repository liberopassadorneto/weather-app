package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

type CepResponse struct {
	City  string `json:"localidade"`
	Error bool   `json:"erro,omitempty"`
}

type WeatherResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type WeatherResult struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func main() {
	http.HandleFunc("/weather", weatherHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	zipcode := r.URL.Query().Get("cep")
	valid, _ := regexp.MatchString(`^\d{8}$`, zipcode)
	if !valid {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"message": "invalid zipcode"})
		return
	}
	cepURL := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", zipcode)
	client := &http.Client{Timeout: 100 * time.Second}
	resp, err := client.Get(cepURL)
	if err != nil {
		http.Error(w, "error querying CEP", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "error reading CEP response", http.StatusInternalServerError)
		return
	}
	var cepData CepResponse
	if err := json.Unmarshal(body, &cepData); err != nil {
		http.Error(w, "error processing CEP response", http.StatusInternalServerError)
		return
	}
	if cepData.Error || cepData.City == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "can not find zipcode"})
		return
	}
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		http.Error(w, "WeatherAPI key not configured", http.StatusInternalServerError)
		return
	}
	encodedCity := url.QueryEscape(cepData.City)
	weatherURL := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", weatherAPIKey, encodedCity)
	weatherResp, err := client.Get(weatherURL)
	if err != nil {
		http.Error(w, "error querying WeatherAPI", http.StatusInternalServerError)
		return
	}
	defer weatherResp.Body.Close()
	weatherBody, err := io.ReadAll(weatherResp.Body)
	if err != nil {
		http.Error(w, "error reading WeatherAPI response", http.StatusInternalServerError)
		return
	}
	var weatherData WeatherResponse
	if err := json.Unmarshal(weatherBody, &weatherData); err != nil {
		http.Error(w, "error processing WeatherAPI response", http.StatusInternalServerError)
		return
	}
	tempC := weatherData.Current.TempC
	tempF := tempC*1.8 + 32
	tempK := tempC + 273
	result := WeatherResult{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
