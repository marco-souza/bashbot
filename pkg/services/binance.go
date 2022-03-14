package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/marco-souza/bashbot/pkg/config"
	"github.com/marco-souza/bashbot/pkg/entities"
)

func FetchAccountSnapshot(walletType string) *entities.AccountSnapshotResponse {
	accountSnapURL := getBinanceEndpoint("account-snapshot")

	params := url.Values{}
	params.Add("type", walletType)
	params.Add("endTime", fmt.Sprint(time.Now().Unix()*1000))

	req := makeSignedRequest(accountSnapURL, params)
	responseBody := Fetch(req)

	var accountSnapshot entities.AccountSnapshotResponse
	if err := json.Unmarshal(responseBody, &accountSnapshot); err != nil {
		panic(err)
	}

	return &accountSnapshot
}

func FetchTicker(currencyPair string) *entities.Ticker {
	// API ref: https://binance-docs.github.io/apidocs/spot/en/#symbol-price-ticker
	tickerURL := getBinanceEndpoint("ticker")

	params := url.Values{}
	params.Add("symbol", currencyPair)

	req := MakeRequest(tickerURL, params)
	responseBody := Fetch(req)

	var ticker entities.Ticker
	if err := json.Unmarshal(responseBody, &ticker); err != nil {
		panic(err)
	}

	log.Println("Tiker", ticker)

	return &ticker
}

type CurrencyPair string

type BinanceWsMessage struct {
	Id     int      `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

func (msg *BinanceWsMessage) Bytes() []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Fatal("msg parsing", err)
	}
	return data
}

type BinanceWsResponse struct {
	Id     int      `json:"id"`
	Result []string `json:"result,omitempty"`
}

func NewResponse(data []byte) BinanceWsResponse {
	var resp BinanceWsResponse
	err := json.Unmarshal(data, &resp)
	if err != nil {
		log.Fatal("msg parsing", err)
	}
	return resp
}

type Candle struct {
	StartTime    int     `json:"t"`
	CloseTime    int     `json:"T"`
	NumTrades    int     `json:"n"`
	IsClosed     bool    `json:"x"`
	Symbol       string  `json:"s"`
	Interval     string  `json:"i"`
	OpenPrice    float64 `json:"o,string"`
	ClosePrice   float64 `json:"c,string"`
	HighPrice    float64 `json:"h,string"`
	LowPrice     float64 `json:"l,string"`
	QuoteVolume  float64 `json:"q,string"`
	BaseVolume   float64 `json:"v,string"`
	LastTradeId  int     `json:"L"`
	FirstTradeId int     `json:"f"`
}

type BaseMessageEvent struct {
	EventTime int    `json:"E"`
	EventType string `json:"e"`
	Symbol    string `json:"s"`
}

type CandleEvent struct {
	*BaseMessageEvent
	CandleData Candle `json:"k"`
}

func WatchSymbol(symbol CurrencyPair, period string, done *chan bool) chan Candle {
	ws := getWsConnection()
	candleChan := make(chan Candle)

	// subscripte to currency pair
	subscribeMsg := BinanceWsMessage{
		Id:     1,
		Method: "SUBSCRIBE",
		Params: []string{
			fmt.Sprintf("%s@kline_%s", symbol, period),
		},
	}
	if err := ws.WriteMessage(websocket.TextMessage, subscribeMsg.Bytes()); err != nil {
		log.Fatal("subscribe: ", err)
	}

	// await response
	_, msg, err := ws.ReadMessage()
	if err != nil {
		log.Println("read: ", err)
	}

	// validate resp
	resp := NewResponse(msg)
	if resp.Id != subscribeMsg.Id {
		log.Fatal("resp: ", resp)
	}

	messageListener := func() {
		// event loop
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				log.Println("read: ", err)
				continue
			}

			var event CandleEvent
			if err := json.Unmarshal(msg, &event); err != nil {
				log.Println("parse: ", err)
			}

			// select event
			switch event.EventType {
			case "kline":
				candleChan <- event.CandleData
			}
		}
	}
	go messageListener()
	go func() {
		<-*done
		close(candleChan)
		ws.Close()
	}()

	return candleChan
}

var conn *websocket.Conn

func getWsConnection() *websocket.Conn {
	addr := url.URL{Scheme: "wss", Host: "stream.binance.com:9443", Path: "/ws"}
	log.Printf("connecting to %s\n", addr.String())

	conn, _, err := websocket.DefaultDialer.Dial(addr.String(), nil)
	if err != nil {
		log.Fatal("dial: ", err)
	}

	return conn
}

var endpoint = map[string]string{
	"account-snapshot": "/sapi/v1/accountSnapshot",
	"system-status":    "/sapi/v1/system/status",
	"ticker":           "/api/v3/ticker/price",
}

func getBinanceEndpoint(name string) string {
	url := endpoint[name]
	if url == "" {
		panic(fmt.Sprintf("No '%s' endpoint found", name))
	}
	return config.BASE_BINANCE_URL + url
}

func makeSignedRequest(url string, params url.Values) *http.Request {
	signedParams := signParams(params)
	req := MakeRequest(url, signedParams)

	req.Header.Set("X-MBX-APIKEY", config.BINANCE_API_KEY)

	return req
}

func signParams(params url.Values) url.Values {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params.Add("timestamp", timestamp)

	signature := sign(params.Encode())
	params.Add("signature", signature)

	return params
}

func sign(text string) string {
	hash := hmac.New(sha256.New, []byte(config.BINANCE_API_SECRET))
	hash.Write([]byte(text))
	return hex.EncodeToString(hash.Sum(nil))
}
