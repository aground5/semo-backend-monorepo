package toss

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/provider"
	"go.uber.org/zap"
)

// IssueBillingKey issues a billing key from Toss Payments
// POST /v1/billing/authorizations/issue
func (t *TossProvider) IssueBillingKey(ctx context.Context, req *provider.IssueBillingKeyRequest) (*provider.IssueBillingKeyResponse, error) {
	t.logger.Info("TossProvider: Issuing billing key - DEBUG",
		zap.String("customer_key", req.CustomerKey),
		zap.String("auth_key_full", req.AuthKey),
		zap.Int("auth_key_length", len(req.AuthKey)))

	body := map[string]string{
		"authKey":     req.AuthKey,
		"customerKey": req.CustomerKey,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "MARSHAL_ERROR",
			Message: "Failed to prepare request",
			Details: err.Error(),
		}
	}

	t.logger.Info("TossProvider: Request body prepared",
		zap.String("request_body", string(jsonBody)))

	url := fmt.Sprintf("%s/%s/billing/authorizations/issue", tossAPIBaseURL, tossAPIVersion)
	t.logger.Info("TossProvider: Calling Toss API",
		zap.String("url", url))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "REQUEST_ERROR",
			Message: "Failed to create request",
			Details: err.Error(),
		}
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.secretKey + ":"))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		t.logger.Error("TossProvider: Billing key issue request failed", zap.Error(err))
		return nil, &provider.ProviderError{
			Code:    "API_ERROR",
			Message: "TossPayments API request failed",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "RESPONSE_ERROR",
			Message: "Failed to read response",
			Details: err.Error(),
		}
	}

	t.logger.Info("TossProvider: Received response from Toss",
		zap.Int("status_code", resp.StatusCode),
		zap.String("response_body", string(respBody)))

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.Unmarshal(respBody, &errResp)

		t.logger.Error("TossProvider: Billing key issue failed - DETAILED DEBUG",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(respBody)),
			zap.String("auth_key_sent", req.AuthKey),
			zap.String("customer_key_sent", req.CustomerKey))

		code, _ := errResp["code"].(string)
		message, _ := errResp["message"].(string)

		return nil, &provider.ProviderError{
			Code:    code,
			Message: message,
			Details: string(respBody),
		}
	}

	var result provider.IssueBillingKeyResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &provider.ProviderError{
			Code:    "PARSE_ERROR",
			Message: "Failed to parse response",
			Details: err.Error(),
		}
	}

	t.logger.Info("TossProvider: Billing key issued successfully",
		zap.String("customer_key", result.CustomerKey),
		zap.String("card_company", result.CardCompany))

	return &result, nil
}

// ChargeBillingKey charges a billing key
// POST /v1/billing/{billingKey}
func (t *TossProvider) ChargeBillingKey(ctx context.Context, req *provider.ChargeBillingKeyRequest) (*provider.ChargeBillingKeyResponse, error) {
	t.logger.Info("TossProvider: Charging billing key",
		zap.String("order_id", req.OrderID),
		zap.Int64("amount", req.Amount))

	body := map[string]interface{}{
		"customerKey": req.CustomerKey,
		"amount":      req.Amount,
		"orderId":     req.OrderID,
		"orderName":   req.OrderName,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "MARSHAL_ERROR",
			Message: "Failed to prepare request",
			Details: err.Error(),
		}
	}

	url := fmt.Sprintf("%s/%s/billing/%s", tossAPIBaseURL, tossAPIVersion, req.BillingKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "REQUEST_ERROR",
			Message: "Failed to create request",
			Details: err.Error(),
		}
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.secretKey + ":"))
	httpReq.Header.Set("Authorization", "Basic "+auth)
	httpReq.Header.Set("Content-Type", "application/json")

	// Billing charges can take up to 60 seconds
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.logger.Error("TossProvider: Billing charge request failed", zap.Error(err))
		return nil, &provider.ProviderError{
			Code:    "API_ERROR",
			Message: "TossPayments API request failed",
			Details: err.Error(),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.ProviderError{
			Code:    "RESPONSE_ERROR",
			Message: "Failed to read response",
			Details: err.Error(),
		}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.Unmarshal(respBody, &errResp)

		t.logger.Error("TossProvider: Billing charge failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(respBody)))

		code, _ := errResp["code"].(string)
		message, _ := errResp["message"].(string)

		return nil, &provider.ProviderError{
			Code:    code,
			Message: message,
			Details: string(respBody),
		}
	}

	var tossResp map[string]interface{}
	if err := json.Unmarshal(respBody, &tossResp); err != nil {
		return nil, &provider.ProviderError{
			Code:    "PARSE_ERROR",
			Message: "Failed to parse response",
			Details: err.Error(),
		}
	}

	result := &provider.ChargeBillingKeyResponse{
		PaymentKey: getStringFromMap(tossResp, "paymentKey"),
		OrderID:    getStringFromMap(tossResp, "orderId"),
		Status:     getStringFromMap(tossResp, "status"),
	}

	if amount, ok := tossResp["totalAmount"].(float64); ok {
		result.Amount = int64(amount)
	}

	if txKey := getStringFromMap(tossResp, "transactionKey"); txKey != "" {
		result.TransactionKey = txKey
	}

	if approvedAt := getStringFromMap(tossResp, "approvedAt"); approvedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, approvedAt); err == nil {
			result.ApprovedAt = &parsed
		}
	}

	t.logger.Info("TossProvider: Billing charge successful",
		zap.String("order_id", result.OrderID),
		zap.String("payment_key", result.PaymentKey))

	return result, nil
}
