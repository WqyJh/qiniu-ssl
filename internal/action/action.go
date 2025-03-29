package action

import (
	"fmt"
	"log"

	"github.com/WqyJh/qiniu-ssl/internal/certmanager"
	"github.com/WqyJh/qiniu-ssl/internal/qiniuapi"
)

func Run(
	qiniuAccessKey, qiniuSecretKey, aliyunAccessKey, aliyunSecretKey, domain, email, certDir, aliyunRegion string,
	forceHTTPS, http2 bool,
) error {
	// Validate required parameters
	if qiniuAccessKey == "" || qiniuSecretKey == "" {
		return fmt.Errorf("qiniu access key and secret key are required")
	}

	if aliyunAccessKey == "" || aliyunSecretKey == "" {
		return fmt.Errorf("aliyun access key and secret key are required")
	}

	if domain == "" {
		return fmt.Errorf("domain name is required")
	}

	// Create certificate manager
	cm, err := certmanager.NewCertManager(domain, email, certDir)
	if err != nil {
		return fmt.Errorf("failed to create certificate manager: %v", err)
	}

	// Request certificate using Aliyun DNS challenge
	log.Printf("Requesting certificate for %s using Aliyun DNS challenge...", domain)
	if err := cm.RequestCertificate(aliyunAccessKey, aliyunSecretKey, aliyunRegion); err != nil {
		return fmt.Errorf("failed to request certificate: %v", err)
	}
	log.Printf("Certificate for %s has been obtained successfully", domain)

	// Get certificate paths
	certPath, keyPath := cm.GetCertificatePaths()
	log.Printf("Certificate saved at: %s", certPath)
	log.Printf("Private key saved at: %s", keyPath)

	// Load certificate
	certPEM, keyPEM, err := cm.LoadCertificate()
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}

	// Create Qiniu client
	qiniu, err := qiniuapi.NewQiniuClient(qiniuAccessKey, qiniuSecretKey)
	if err != nil {
		return fmt.Errorf("failed to create Qiniu client: %v", err)
	}

	// Upload certificate to Qiniu
	log.Printf("Uploading certificate to Qiniu...")
	certID, err := qiniu.UploadCertificate(domain, certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to upload certificate: %v", err)
	}
	log.Printf("Certificate has been uploaded to Qiniu with ID: %s", certID)

	// Get domain information to check if HTTPS is supported
	log.Printf("Retrieving domain information for %s...", domain)
	domainInfo, err := qiniu.GetDomainInfo(domain)
	if err != nil {
		return fmt.Errorf("failed to retrieve domain information: %v", err)
	}

	// Check if HTTPS is supported - use the HTTPS field instead of Protocol
	httpsSupported := domainInfo.HTTPS != nil && domainInfo.HTTPS.CertID != ""

	// Enable HTTPS if not supported
	if !httpsSupported {
		log.Printf("Domain %s does not support HTTPS, enabling it now...", domain)
		// 调用 sslize 接口启用 HTTPS，并直接绑定证书
		if err := qiniu.EnableHTTPS(domain, certID, forceHTTPS, http2); err != nil {
			return fmt.Errorf("failed to enable HTTPS for domain: %v", err)
		}
		log.Printf("HTTPS has been enabled for domain %s with certificate ID %s", domain, certID)
	} else {
		log.Printf("Domain %s already supports HTTPS, updating certificate...", domain)
		// Update HTTPS configuration with the new certificate
		log.Printf("Updating HTTPS configuration for domain %s...", domain)
		if err := qiniu.UpdateHTTPSConfig(domain, certID, forceHTTPS, http2); err != nil {
			return fmt.Errorf("failed to update HTTPS configuration: %v", err)
		}
		log.Printf("HTTPS configuration has been updated for domain %s successfully", domain)
	}

	log.Printf("All operations completed successfully!")
	return nil
}
