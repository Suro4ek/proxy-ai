package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type APIConfig struct {
	DefaultTargetURL string
}

func main() {
	port := flag.Int("port", 8080, "Порт для прослушивания")
	flag.Parse()

	apiConfigs := map[string]APIConfig{
		"openai": {
			DefaultTargetURL: "https://api.openai.com",
		},
		"claude": {
			DefaultTargetURL: "https://api.anthropic.com",
		},
	}

	log.Printf("Запуск прокси-сервера на порту %d", *port)
	for apiName, config := range apiConfigs {
		log.Printf("Настроен прокси для %s: %s по умолчанию",
			apiName, config.DefaultTargetURL)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		pathParts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 3)
		if len(pathParts) < 3 || pathParts[0] != "proxy" {
			http.Error(w, "Неверный формат пути. Используйте /proxy/{api-name}/...", http.StatusBadRequest)
			return
		}

		apiName := pathParts[1]
		apiPath := pathParts[2]

		apiConfig, exists := apiConfigs[apiName]
		if !exists {
			http.Error(w, fmt.Sprintf("Неподдерживаемый API: %s", apiName), http.StatusBadRequest)
			return
		}

		targetURL, err := url.Parse(fmt.Sprintf("%s/%s", apiConfig.DefaultTargetURL, apiPath))
		if err != nil {
			http.Error(w, "Ошибка при создании целевого URL: "+err.Error(), http.StatusInternalServerError)
			return
		}

		targetURL.RawQuery = r.URL.RawQuery

		proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
		if err != nil {
			http.Error(w, "Ошибка при создании запроса: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Копируем заголовки запроса (кроме заголовков с ключами API и серверами)
		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(header, value)
			}

		}

		// proxyReq.Header.Set("X-Forwarded-Host", r.Host)
		// proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

		client := &http.Client{}
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "Ошибка при отправке запроса: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer proxyResp.Body.Close()

		log.Printf("%s %s -> %s %d", r.Method, r.URL.Path, targetURL.String(), proxyResp.StatusCode)

		for header, values := range proxyResp.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}

		w.WriteHeader(proxyResp.StatusCode)

		io.Copy(w, proxyResp.Body)
	})

	// Запускаем сервер
	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}
}
