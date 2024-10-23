package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type WSRequest struct {
	Method int         `json:"method,omitempty"`
	Params interface{} `json:"params,omitempty"`
	ID     int         `json:"id"`
}

type AuthParams struct {
	Token string `json:"token"`
}

type AuthResponse struct {
	ID     int `json:"id"`
	Result struct {
		Client  string `json:"client"`
		Version string `json:"version"`
		Expires bool   `json:"expires"`
		TTL     int    `json:"ttl"`
	} `json:"result"`
}

type SubscribeParams struct {
	Channel string `json:"channel"`
}

type WSError struct {
	Reason    string `json:"reason"`
	Reconnect bool   `json:"reconnect"`
}

type ChartTickResponse struct {
	Result struct {
		Channel string `json:"channel"`
		Data    struct {
			Data [][]interface{} `json:"data"`
		} `json:"data"`
	} `json:"result"`
}

type WSClient struct {
	conn *websocket.Conn
}

func NewWSClient() (*WSClient, error) {
	u := url.URL{Scheme: "wss", Host: "ws3.indodax.com", Path: "/ws/"}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return &WSClient{conn: conn}, nil
}

func (c *WSClient) authenticate() error {
	authReq := WSRequest{
		Params: AuthParams{
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE5NDY2MTg0MTV9.UR1lBM6Eqh0yWz-PVirw1uPCxe60FdchR8eNVdsskeo",
		},
		ID: 1,
	}

	if err := c.conn.WriteJSON(authReq); err != nil {
		return err
	}

	var response AuthResponse
	if err := c.conn.ReadJSON(&response); err != nil {
		return err
	}

	log.Printf("Authentication successful. Client ID: %s", response.Result.Client)
	return nil
}

func (c *WSClient) pingPong() error {
	pingReq := WSRequest{
		Method: 7,
		ID:     3,
	}

	if err := c.conn.WriteJSON(pingReq); err != nil {
		return err
	}

	var response map[string]interface{}
	if err := c.conn.ReadJSON(&response); err != nil {
		return err
	}

	return nil
}

func (c *WSClient) subscribe(channel string) error {
	subReq := WSRequest{
		Method: 1,
		Params: SubscribeParams{Channel: channel},
		ID:     2,
	}

	if err := c.conn.WriteJSON(subReq); err != nil {
		return err
	}

	var response map[string]interface{}
	if err := c.conn.ReadJSON(&response); err != nil {
		if closeErr, ok := err.(*websocket.CloseError); ok {
			var wsErr WSError
			if jsonErr := json.Unmarshal([]byte(closeErr.Text), &wsErr); jsonErr == nil {
				return fmt.Errorf("subscription failed: %s (reconnect: %v)", wsErr.Reason, wsErr.Reconnect)
			}
		}
		return err
	}

	log.Printf("Subscribed to channel: %s", channel)
	return nil
}

func formatNumber(num float64) string {
	return fmt.Sprintf("%,.0f", num)
}

func (c *WSClient) handleMessages() {
	for {
		var tickResponse ChartTickResponse
		err := c.conn.ReadJSON(&tickResponse)
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				var wsErr WSError
				if jsonErr := json.Unmarshal([]byte(closeErr.Text), &wsErr); jsonErr == nil {
					log.Printf("WebSocket closed: %s (reconnect: %v)", wsErr.Reason, wsErr.Reconnect)
					return
				}
			}
			log.Printf("Error reading message: %v", err)
			return
		}

		for _, data := range tickResponse.Result.Data.Data {
			if len(data) >= 4 {
				timestamp := int64(data[0].(float64))
				price := data[2].(float64)
				volume := data[3].(string)

				date := time.Unix(timestamp, 0).Format("02-01-2006 15:04:05")

				fmt.Print("\033[H\033[2J")

				fmt.Println("╔════════════════════════════════════════╗")
				fmt.Println("║           BTC/IDR LIVE PRICE           ║")
				fmt.Println("╠════════════════════════════════════════╣")
				fmt.Printf("║ DATE   : %s         ║\n", date)
				fmt.Printf("║ PRICE  : Rp %s           ║\n", formatNumber(price))
				fmt.Printf("║ VOLUME : %s BTC              ║\n", volume)
				fmt.Println("╚════════════════════════════════════════╝")
			}
		}
	}
}

func main() {
	client, err := NewWSClient()
	if err != nil {
		log.Fatal("Error creating WebSocket client:", err)
	}
	defer client.conn.Close()

	if err := client.authenticate(); err != nil {
		log.Fatal("Authentication failed:", err)
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			if err := client.pingPong(); err != nil {
				log.Printf("Ping failed: %v", err)
			}
		}
	}()

	if err := client.subscribe("chart:tick-btcidr"); err != nil {
		log.Fatal("Subscription failed:", err)
	}

	go client.handleMessages()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down WebSocket client...")
	ticker.Stop()
}
