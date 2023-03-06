package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"bytes"
)

func main() {

	e := echo.New()

	// Set up CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))

	// Proxy handler
	e.Any("/*", func(c echo.Context) error {
		req := c.Request()
		fmt.Println("proxy hit")

		// Print the request details
		fmt.Printf("Method: %s\n", req.Method)
		fmt.Printf("URL: %s\n", req.URL)
		fmt.Printf("Headers: %v\n", req.Header)

		// Make request to target server
		client := &http.Client{}
		targetURL := "https://api.openai.com" + req.URL.Path
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		defer req.Body.Close()

		targetReq, err := http.NewRequest(req.Method, targetURL, ioutil.NopCloser(
			bytes.NewReader(body),
		))

		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		targetReq.Header = make(http.Header)
		for k, v := range req.Header {
			targetReq.Header.Set(k, v[0])
		}

		// Get Authorization header from incoming request
		authHeader := req.Header.Get("Authorization")

		// Add Authorization header to target server request
		targetReq.Header.Set("Authorization", authHeader)

		targetResp, err := client.Do(targetReq)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		defer targetResp.Body.Close()

		// Copy headers from target server response to proxy response
		for k, v := range targetResp.Header {
			for _, vv := range v {
				c.Response().Header().Add(k, vv)
			}
		}

		c.Response().WriteHeader(targetResp.StatusCode)
		// Read the response body and unmarshal it into a JSON object
		respBody, err := ioutil.ReadAll(targetResp.Body)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		var respJSON map[string]interface{}
		err = json.Unmarshal(respBody, &respJSON)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		// Print the JSON object
		fmt.Println(respJSON)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		_, err = c.Response().Write(respBody)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		return nil
	})

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
