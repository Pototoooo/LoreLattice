package billing

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Public-key mode, RSA2, direct merchant PC website payment. The gateway is
// deliberately fixed to production: sandbox credentials cannot credit real wallets.
type AlipayClient struct {
	appID, sellerID, notifyURL, returnURL string
	privateKey                            *rsa.PrivateKey
	publicKey                             *rsa.PublicKey
	httpClient                            *http.Client
	gateway                               string
}

func NewAlipayClientFromEnv() (*AlipayClient, error) {
	if !envBool("ALIPAY_ENABLED") {
		return nil, nil
	}
	c := &AlipayClient{appID: os.Getenv("ALIPAY_APP_ID"), sellerID: os.Getenv("ALIPAY_SELLER_ID"), notifyURL: os.Getenv("ALIPAY_NOTIFY_URL"), returnURL: os.Getenv("ALIPAY_RETURN_URL"), gateway: "https://openapi.alipay.com/gateway.do", httpClient: &http.Client{Timeout: 10 * time.Second}}
	if c.appID == "" || c.sellerID == "" {
		return nil, fmt.Errorf("ALIPAY_APP_ID and ALIPAY_SELLER_ID required")
	}
	for _, raw := range []string{c.notifyURL, c.returnURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
			return nil, fmt.Errorf("Alipay notification and return URLs must be public HTTPS URLs")
		}
	}
	privatePEM, err := os.ReadFile(os.Getenv("ALIPAY_PRIVATE_KEY_FILE"))
	if err != nil {
		return nil, fmt.Errorf("read Alipay merchant private key: %w", err)
	}
	publicPEM, err := os.ReadFile(os.Getenv("ALIPAY_PUBLIC_KEY_FILE"))
	if err != nil {
		return nil, fmt.Errorf("read Alipay public key: %w", err)
	}
	block, _ := pem.Decode(privatePEM)
	if block == nil {
		return nil, fmt.Errorf("invalid merchant PEM private key")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		c.privateKey, _ = key.(*rsa.PrivateKey)
	} else {
		c.privateKey, _ = x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	block, _ = pem.Decode(publicPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid Alipay PEM public key")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid Alipay public key: %w", err)
	}
	c.publicKey, _ = key.(*rsa.PublicKey)
	if c.privateKey == nil || c.publicKey == nil || c.privateKey.N.BitLen() < 2048 || c.publicKey.N.BitLen() < 2048 {
		return nil, fmt.Errorf("Alipay requires RSA keys of at least 2048 bits")
	}
	return c, nil
}
func alipayCanonical(values url.Values, notification bool) (string, error) {
	keys := make([]string, 0, len(values))
	for k, v := range values {
		if len(v) != 1 {
			return "", fmt.Errorf("duplicate payment parameter")
		}
		if k == "sign" || (notification && k == "sign_type") || v[0] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+values.Get(k))
	}
	return strings.Join(parts, "&"), nil
}
func rsaSign(key *rsa.PrivateKey, message []byte) (string, error) {
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	return base64.StdEncoding.EncodeToString(signature), err
}
func rsaVerify(key *rsa.PublicKey, message []byte, signature string) error {
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(message)
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], decoded)
}
func (c *AlipayClient) params(method string, biz any) (url.Values, error) {
	raw, err := json.Marshal(biz)
	if err != nil {
		return nil, err
	}
	return url.Values{"app_id": {c.appID}, "method": {method}, "format": {"JSON"}, "charset": {"utf-8"}, "sign_type": {"RSA2"}, "timestamp": {time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")}, "version": {"1.0"}, "biz_content": {string(raw)}}, nil
}
func (c *AlipayClient) sign(values url.Values) error {
	canonical, err := alipayCanonical(values, false)
	if err != nil {
		return err
	}
	signature, err := rsaSign(c.privateKey, []byte(canonical))
	if err == nil {
		values.Set("sign", signature)
	}
	return err
}
func (c *AlipayClient) Checkout(order PaymentOrder) (string, error) {
	values, err := c.params("alipay.trade.page.pay", map[string]any{
		"out_trade_no": order.ID, "total_amount": formatFen(order.AmountFen), "subject": "LoreLattice AI 使用余额充值", "product_code": "FAST_INSTANT_TRADE_PAY",
		"time_expire": order.ExpiresAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05"),
	})
	if err != nil {
		return "", err
	}
	values.Set("notify_url", c.notifyURL)
	values.Set("return_url", c.returnURL)
	if err := c.sign(values); err != nil {
		return "", err
	}
	return c.gateway + "?" + values.Encode(), nil
}

type PaymentConfirmation struct {
	OrderID, TradeID, AppID, SellerID, Status string
	AmountFen                                 int64
}

func (c *AlipayClient) VerifyNotification(values url.Values) (*PaymentConfirmation, error) {
	if values.Get("sign_type") != "RSA2" {
		return nil, fmt.Errorf("unsupported signature algorithm")
	}
	canonical, err := alipayCanonical(values, true)
	if err != nil {
		return nil, err
	}
	if err := rsaVerify(c.publicKey, []byte(canonical), values.Get("sign")); err != nil {
		return nil, fmt.Errorf("payment signature verification failed")
	}
	amount, err := parseFen(values.Get("total_amount"))
	if err != nil {
		return nil, err
	}
	if values.Get("app_id") != c.appID || values.Get("seller_id") != c.sellerID {
		return nil, fmt.Errorf("payment merchant mismatch")
	}
	return &PaymentConfirmation{OrderID: values.Get("out_trade_no"), TradeID: values.Get("trade_no"), AppID: values.Get("app_id"), SellerID: values.Get("seller_id"), Status: values.Get("trade_status"), AmountFen: amount}, nil
}
func (c *AlipayClient) Query(ctx context.Context, order PaymentOrder) (*PaymentConfirmation, error) {
	if order.AppID != c.appID || order.SellerID != c.sellerID {
		return nil, fmt.Errorf("payment query merchant configuration mismatch")
	}
	values, err := c.params("alipay.trade.query", map[string]string{"out_trade_no": order.ID})
	if err != nil {
		return nil, err
	}
	if err := c.sign(values); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.gateway, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("payment query returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Response json.RawMessage `json:"alipay_trade_query_response"`
		Sign     string          `json:"sign"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	// Verify the exact signed response bytes, not reserialized JSON.
	// trade.query does not guarantee seller_id in the response. Merchant scope
	// comes from our signed app request and immutable order identity; if present,
	// the returned seller_id must additionally match. Notifications require it.
	if err := rsaVerify(c.publicKey, envelope.Response, envelope.Sign); err != nil {
		return nil, fmt.Errorf("invalid payment query signature")
	}
	var result struct {
		Code     string `json:"code"`
		OrderID  string `json:"out_trade_no"`
		TradeID  string `json:"trade_no"`
		SellerID string `json:"seller_id"`
		Status   string `json:"trade_status"`
		Amount   string `json:"total_amount"`
	}
	if err := json.Unmarshal(envelope.Response, &result); err != nil {
		return nil, err
	}
	if result.Code != "10000" {
		return nil, fmt.Errorf("payment query not successful: %s", result.Code)
	}
	if result.OrderID != order.ID || (result.SellerID != "" && result.SellerID != c.sellerID) {
		return nil, fmt.Errorf("payment query identity mismatch")
	}
	amount, err := parseFen(result.Amount)
	if err != nil {
		return nil, err
	}
	return &PaymentConfirmation{OrderID: result.OrderID, TradeID: result.TradeID, AppID: c.appID, SellerID: c.sellerID, Status: result.Status, AmountFen: amount}, nil
}
func formatFen(fen int64) string { return fmt.Sprintf("%d.%02d", fen/100, fen%100) }
func parseFen(raw string) (int64, error) {
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts[0]) > 9 {
		return 0, fmt.Errorf("invalid payment amount")
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid payment amount")
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, fmt.Errorf("invalid payment precision")
		}
		fraction = parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		for _, r := range fraction {
			if r < '0' || r > '9' {
				return 0, fmt.Errorf("invalid payment amount")
			}
		}
	}
	cents, _ := strconv.ParseInt(fraction, 10, 64)
	if whole*100+cents <= 0 {
		return 0, fmt.Errorf("invalid payment amount")
	}
	return whole*100 + cents, nil
}
