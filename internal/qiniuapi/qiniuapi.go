package qiniuapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
)

const (
	// QiniuAPIHost is the Qiniu API host
	QiniuAPIHost = "https://api.qiniu.com"
)

// QiniuClient represents a Qiniu API client
type QiniuClient struct {
	accessKey string
	secretKey string
	mac       *auth.Credentials
	client    *http.Client
}

// CertificateInfo represents certificate information
type CertificateInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Common      string   `json:"common_name"`
	DNSNames    []string `json:"dnsnames"`
	NotBefore   int64    `json:"not_before"`
	NotAfter    int64    `json:"not_after"`
	CreateTime  int64    `json:"create_time"`
	Description string   `json:"description"`
	Pri         string   `json:"pri,omitempty"`
	Ca          string   `json:"ca,omitempty"`
}

type CertificateResponse struct {
	Code  int             `json:"code"`
	Error string          `json:"error"`
	Cert  CertificateInfo `json:"cert"`
}

// CertificateUploadRequest represents a certificate upload request
type CertificateUploadRequest struct {
	Name        string `json:"name"`
	Common_name string `json:"common_name"`
	Ca          string `json:"ca"`
	Pri         string `json:"pri"`
	Description string `json:"description,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Code    int             `json:"code"`
	Error   string          `json:"error"`
	Result  json.RawMessage `json:"result"`
	Message string          `json:"message"`
}

// DomainInfo represents domain configuration information
type DomainInfo struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Platform string      `json:"platform"`
	GeoCover string      `json:"geoCover"`
	Protocol string      `json:"protocol"`
	HTTPS    *HTTPSInfo  `json:"https,omitempty"`
	Source   interface{} `json:"source"`
}

// HTTPSInfo represents HTTPS configuration information
type HTTPSInfo struct {
	CertID      string `json:"certid"`
	ForceHttps  bool   `json:"forceHttps"`
	Http2Enable bool   `json:"http2Enable"`
}

// NewQiniuClient creates a new Qiniu API client
func NewQiniuClient(accessKey, secretKey string) (*QiniuClient, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("access key and secret key cannot be empty")
	}

	mac := auth.New(accessKey, secretKey)
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &QiniuClient{
		accessKey: accessKey,
		secretKey: secretKey,
		mac:       mac,
		client:    client,
	}, nil
}

// doRequest performs an API request
func (q *QiniuClient) doRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Sign the request
	token, err := q.mac.SignRequestV2(req)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Qiniu "+token)

	// Send the request
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("API error: %s", respBody)
		}
		log.Printf("API error: %s", respBody)
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	return respBody, nil
}

// UploadCertificate uploads a SSL certificate to Qiniu
func (q *QiniuClient) UploadCertificate(name string, certPEM, keyPEM []byte) (string, error) {
	ctx := context.Background()

	// Configure certificate upload
	certConfig := CertificateUploadRequest{
		Name:        name,
		Common_name: name,
		Pri:         string(keyPEM),
		Ca:          string(certPEM),
	}

	// Serialize request
	reqBody, err := json.Marshal(certConfig)
	if err != nil {
		return "", err
	}

	// Make API request
	// 根据七牛云Python SDK的实现，证书上传API的正确路径为 /sslcert
	url := QiniuAPIHost + "/sslcert"
	respBody, err := q.doRequest(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to upload certificate: %v", err)
	}

	// Parse response
	var apiResp struct {
		CertID string `json:"certID"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return apiResp.CertID, nil
}

// GetDomainInfo retrieves information about a CDN domain
func (q *QiniuClient) GetDomainInfo(domain string) (*DomainInfo, error) {
	ctx := context.Background()

	// Build request URL
	url := fmt.Sprintf("%s/domain/%s", QiniuAPIHost, domain)
	respBody, err := q.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain info: %v", err)
	}

	// Parse response
	var info DomainInfo
	if err := json.Unmarshal(respBody, &info); err != nil {
		return nil, fmt.Errorf("failed to parse domain info: %v", err)
	}

	return &info, nil
}

// UpdateHTTPSConfig updates the HTTPS configuration for a domain with a certificate
// This can be used to enable HTTPS support for a domain that doesn't already have it
// or to update an existing HTTPS configuration with a new certificate
func (q *QiniuClient) UpdateHTTPSConfig(domain, certID string, forceHTTPS, http2Enable bool) error {
	ctx := context.Background()

	httpsConfig := struct {
		CertID      string `json:"certid"`
		ForceHttps  bool   `json:"forceHttps"`
		Http2Enable bool   `json:"http2Enable"`
	}{
		CertID:      certID,
		ForceHttps:  forceHTTPS,
		Http2Enable: http2Enable,
	}

	// Serialize request
	reqBody, err := json.Marshal(httpsConfig)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/domain/%s/httpsconf", QiniuAPIHost, domain)
	_, err = q.doRequest(ctx, http.MethodPut, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to update HTTPS configuration: %v", err)
	}

	return nil
}

// EnableHTTPS enables HTTPS for a domain
// This should be called before UpdateHTTPSConfig if HTTPS is not already enabled
func (q *QiniuClient) EnableHTTPS(domain, certID string, forceHTTPS, http2Enable bool) error {
	ctx := context.Background()

	// API要求使用PUT方法并包含请求体
	httpsConfig := struct {
		CertID      string `json:"certid"`
		ForceHttps  bool   `json:"forceHttps"`
		Http2Enable bool   `json:"http2Enable"`
	}{
		CertID:      certID,
		ForceHttps:  forceHTTPS,
		Http2Enable: http2Enable,
	}

	// Serialize request
	reqBody, err := json.Marshal(httpsConfig)
	if err != nil {
		return err
	}

	// 使用正确的API路径
	url := fmt.Sprintf("%s/domain/%s/sslize", QiniuAPIHost, domain)
	_, err = q.doRequest(ctx, http.MethodPut, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to enable HTTPS for domain: %v", err)
	}

	return nil
}

// GetCertificateInfo retrieves information about a specific certificate by ID
func (q *QiniuClient) GetCertificateInfo(certID string) (*CertificateInfo, error) {
	ctx := context.Background()

	url := fmt.Sprintf("%s/sslcert/%s", QiniuAPIHost, certID)
	respBody, err := q.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate info: %v", err)
	}

	// Parse response
	var cert CertificateResponse

	// First, unmarshal the data to a map to process the date formats
	if err := json.Unmarshal(respBody, &cert); err != nil {
		return nil, fmt.Errorf("failed to parse certificate info: %v", err)
	}

	return &cert.Cert, nil
}

// CheckCertificateFromQiniu checks if a domain's certificate in Qiniu is about to expire
func (q *QiniuClient) CheckCertificateFromQiniu(domain string, thresholdDays int) (bool, *CertificateInfo, error) {
	// Get domain information
	domainInfo, err := q.GetDomainInfo(domain)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get domain info: %v", err)
	}

	// Check if HTTPS is enabled and a certificate is bound
	if domainInfo.HTTPS == nil || domainInfo.HTTPS.CertID == "" {
		return true, nil, fmt.Errorf("domain does not have HTTPS enabled or certificate bound")
	}

	// Get certificate information
	certID := domainInfo.HTTPS.CertID
	certInfo, err := q.GetCertificateInfo(certID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get certificate info: %v", err)
	}

	// Check if certificate is about to expire
	thresholdTime := time.Now().AddDate(0, 0, thresholdDays)
	needsRenewal := certInfo.NotAfter < thresholdTime.Unix()

	return needsRenewal, certInfo, nil
}
