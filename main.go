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

	"github.com/gin-gonic/gin"
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
	router := gin.Default()
	router.GET("/weather", weatherHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server running on port %s\n", port)
	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}

func weatherHandler(c *gin.Context) {
	zipcode := c.Query("cep")
	valid, _ := regexp.MatchString(`^\d{8}$`, zipcode)
	if !valid {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "invalid zipcode"})
		return
	}
	encodedZipCode := url.QueryEscape(zipcode)
	cepURL := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", encodedZipCode)
	client := &http.Client{Timeout: 100 * time.Second}
	resp, err := client.Get(cepURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error querying CEP"})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error reading CEP response"})
		return
	}
	var cepData CepResponse
	if err := json.Unmarshal(body, &cepData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error processing CEP response"})
		return
	}
	if cepData.Error || cepData.City == "" {
		c.JSON(http.StatusNotFound, gin.H{"message": "can not find zipcode"})
		return
	}
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "WeatherAPI key not configured"})
		return
	}
	encodedCity := url.QueryEscape(cepData.City)
	weatherURL := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s", weatherAPIKey, encodedCity)
	weatherResp, err := client.Get(weatherURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error querying WeatherAPI"})
		return
	}
	defer weatherResp.Body.Close()
	weatherBody, err := io.ReadAll(weatherResp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error reading WeatherAPI response"})
		return
	}
	var weatherData WeatherResponse
	if err := json.Unmarshal(weatherBody, &weatherData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "error processing WeatherAPI response"})
		return
	}
	tempC := weatherData.Current.TempC
	tempF := tempC*1.8 + 32
	tempK := tempC + 273.15
	result := WeatherResult{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}
	c.JSON(http.StatusOK, result)
}
