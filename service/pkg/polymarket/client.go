package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	sdk "github.com/GoPolymarket/polymarket-go-sdk/v2"
	"github.com/GoPolymarket/polymarket-go-sdk/v2/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/v2/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/v2/pkg/clob/clobtypes"
)

type Position struct {
	Asset       string  `json:"asset"`
	Outcome     string  `json:"outcome"`
	Size        float64 `json:"size"`
	AvgPrice    float64 `json:"avgPrice"`
	CurPrice    float64 `json:"curPrice"`
	Title       string  `json:"title"`
	RealizedPnl float64 `json:"realizedPnl"`
}

type Client struct {
	httpCli  *http.Client
	sdk      *sdk.Client
	signer   auth.Signer
	creds    *auth.APIKey
	canTrade bool
}

func New(privateKey, apiKey, apiSecret, apiPassphrase string) (*Client, error) {
	c := &Client{httpCli: &http.Client{Timeout: 30 * time.Second}}

	sdkClient, err := sdk.NewClientE()
	if err != nil {
		return nil, fmt.Errorf("init sdk: %w", err)
	}
	c.sdk = sdkClient

	if privateKey != "" && apiKey != "" {
		signer, err := auth.NewPrivateKeySigner(privateKey, 137)
		if err != nil {
			return nil, fmt.Errorf("create signer: %w", err)
		}
		c.signer = signer
		c.creds = &auth.APIKey{
			Key:        apiKey,
			Secret:     apiSecret,
			Passphrase: apiPassphrase,
		}
		c.sdk = sdkClient.WithAuth(signer, c.creds)
		c.canTrade = true
	}

	return c, nil
}

func (c *Client) CanTrade() bool {
	return c.canTrade
}

func (c *Client) FetchOpenPositions(ctx context.Context, walletAddr string) ([]Position, error) {
	return c.fetchAllPositions(ctx, "positions", walletAddr)
}

func (c *Client) FetchClosedPositions(ctx context.Context, walletAddr string) ([]Position, error) {
	return c.fetchAllPositions(ctx, "closed-positions", walletAddr)
}

func (c *Client) fetchAllPositions(ctx context.Context, endpoint, walletAddr string) ([]Position, error) {
	const pageSize = 50
	const maxPages = 20

	var all []Position
	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		positions, err := c.fetchPositionsPage(ctx, endpoint, walletAddr, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, positions...)
		if len(positions) < pageSize {
			break
		}
	}
	return all, nil
}

func (c *Client) fetchPositionsPage(ctx context.Context, endpoint, walletAddr string, limit, offset int) ([]Position, error) {
	url := fmt.Sprintf("https://data-api.polymarket.com/%s?user=%s&limit=%d&offset=%d", endpoint, walletAddr, limit, offset)

	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt)*time.Second + time.Duration(attempt*100)*time.Millisecond)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpCli.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("API 429")
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
		}

		var positions []Position
		if err := json.NewDecoder(resp.Body).Decode(&positions); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		return positions, nil
	}
	return nil, fmt.Errorf("rate limited after retries: %w", lastErr)
}

type CopyTrade struct {
	TokenID string
	Price   float64
	Size    float64
	Market  string
	Outcome string
}

func (c *Client) BuildCopyTrade(pos Position, amountUSD float64) CopyTrade {
	price := pos.CurPrice
	if price <= 0 {
		price = pos.AvgPrice
	}
	size := amountUSD / price
	if size < 0.01 {
		size = 0.01
	}

	return CopyTrade{
		TokenID: pos.Asset,
		Price:   price,
		Size:    size,
		Market:  pos.Title,
		Outcome: pos.Outcome,
	}
}

func (c *Client) PlaceOrder(ctx context.Context, trade CopyTrade) error {
	return c.placeOrder(ctx, trade, "BUY")
}

func (c *Client) CloseOrder(ctx context.Context, trade CopyTrade) error {
	return c.placeOrder(ctx, trade, "SELL")
}

func (c *Client) placeOrder(ctx context.Context, trade CopyTrade, side string) error {
	if !c.canTrade {
		return fmt.Errorf("no auth credentials configured")
	}

	order, err := clob.NewOrderBuilder(c.sdk.CLOB, c.signer).
		TokenID(trade.TokenID).
		Side(side).
		Price(trade.Price).
		Size(trade.Size).
		OrderType(clobtypes.OrderTypeGTC).
		Build()
	if err != nil {
		return fmt.Errorf("build order: %w", err)
	}

	resp, err := c.sdk.CLOB.CreateOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	if resp.ID == "" {
		return fmt.Errorf("order rejected: %+v", resp)
	}

	return nil
}

func volumeToFloat(v interface{}) float64 {
	switch val := v.(type) {
	case string:
		f, _, err := big.ParseFloat(val, 10, 64, big.ToNearestEven)
		if err != nil {
			return 0
		}
		fl, _ := f.Float64()
		return fl
	case float64:
		return val
	default:
		return 0
	}
}
