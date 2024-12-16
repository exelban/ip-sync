package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/pkgz/logg"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type app struct {
	Cloudflare struct {
		ApiToken string `long:"api-token" env:"API_TOKEN" description:"cloudflare api token"`
		ZoneID   string `long:"zone-id" env:"ZONE_ID" description:"cloudflare zone id"`
		Record   string `long:"record" env:"RECORD" description:"cloudflare record name"`
		RecordID string
	} `group:"cloudflare" namespace:"cloudflare" env-namespace:"CLOUDFLARE"`

	Interval time.Duration `long:"interval" default:"5m" description:"sync interval"`

	Debug bool `long:"debug" env:"DEBUG" description:"debug mode"`
}

var version = "dev"

func main() {
	fmt.Println(version)

	var application app
	p := flags.NewParser(&application, flags.Default)
	if _, err := p.Parse(); err != nil {
		fmt.Printf("error parse args: %v", err)
		os.Exit(1)
	}

	if application.Cloudflare.ApiToken == "" || application.Cloudflare.ZoneID == "" || application.Cloudflare.Record == "" {
		fmt.Println("cloudflare api token, zone id and record name are required")
		os.Exit(1)
	}

	logg.NewGlobal(os.Stdout)
	if application.Debug {
		logg.DebugMode()
	}

	ctx := context.Background()
	if err := application.sync(ctx); err != nil {
		log.Printf("[ERROR] sync: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(ctx)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	tk := time.NewTicker(application.Interval)
	log.Printf("[INFO] ip sync every %v", application.Interval)

	for {
		select {
		case <-tk.C:
			if err := application.sync(ctx); err != nil {
				log.Printf("[ERROR] %v", err)
			}
		case <-stop:
			log.Print("[INFO] interrupt signal")
			cancel()
			tk.Stop()
			return
		}
	}
}

// sync checks the current public IP address and updates the cloudflare DNS record if necessary.
func (a *app) sync(ctx context.Context) error {
	recordID, currentValue, err := a.getRecord(ctx, a.Cloudflare.Record)
	if err != nil {
		return fmt.Errorf("get record: %w", err)
	}
	if a.Cloudflare.RecordID != recordID {
		a.Cloudflare.RecordID = recordID
	}

	newIP, err := a.currentIP()
	if err != nil {
		return fmt.Errorf("get current ip: %w", err)
	}

	if currentValue != newIP {
		log.Printf("[INFO] update record %s: %s -> %s", a.Cloudflare.Record, currentValue, newIP)
		if err := a.updateDNSRecord(ctx, newIP); err != nil {
			return fmt.Errorf("update record: %w", err)
		}
	}

	return nil
}

// currentIP returns the current public IP address.
func (a *app) currentIP() (string, error) {
	resp, err := http.Get("https://api.serhiy.io/v1/stats/ip")
	if err != nil {
		return "", fmt.Errorf("get ip: %w", err)
	}
	defer resp.Body.Close()

	response := struct {
		IPv4 *string `json:"ipv4,omitempty"`
		IPv6 *string `json:"ipv6,omitempty"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if response.IPv4 != nil {
		return *response.IPv4, nil
	} else if response.IPv6 != nil {
		return *response.IPv6, nil
	}

	return "", fmt.Errorf("no ip address")
}

// updateDNSRecord updates the cloudflare DNS record with the new IP address.
// getRecord returns the cloudflare DNS record ID and the current IP address.
func (a *app) updateDNSRecord(ctx context.Context, newIP string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", a.Cloudflare.ZoneID, a.Cloudflare.RecordID)
	record := map[string]interface{}{
		"type":    "A",
		"name":    a.Cloudflare.Record,
		"content": newIP,
	}
	body, _ := json.Marshal(record)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.Cloudflare.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
func (a *app) getRecord(ctx context.Context, recordName string) (string, string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", a.Cloudflare.ZoneID, recordName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.Cloudflare.ApiToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	if !result.Success {
		return "", "", fmt.Errorf("api error: %v", result.Errors)
	}

	for _, record := range result.Result {
		if record.Name == recordName && record.Type == "A" {
			return record.ID, record.Content, nil
		}
	}

	return "", "", fmt.Errorf("record not found")
}
