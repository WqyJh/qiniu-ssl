package certmanager

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/WqyJh/qiniu-ssl/internal/aliyundns"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// CertManager handles Let's Encrypt SSL certificate operations
type CertManager struct {
	Domain   string
	Email    string
	CacheDir string
	certPath string
	keyPath  string
}

// User implements the registration.User interface
type User struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

// GetEmail returns the email address of the user
func (u *User) GetEmail() string {
	return u.Email
}

// GetRegistration returns the registration resource
func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

// GetPrivateKey returns the private key of the user
func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

// NewCertManager creates a new certificate manager
func NewCertManager(domain, email, cacheDir string) (*CertManager, error) {
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	if cacheDir == "" {
		cacheDir = "certs"
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	cm := &CertManager{
		Domain:   domain,
		Email:    email,
		CacheDir: cacheDir,
		certPath: filepath.Join(cacheDir, domain+".crt"),
		keyPath:  filepath.Join(cacheDir, domain+".key"),
	}

	return cm, nil
}

// RequestCertificate requests a new certificate from Let's Encrypt using DNS-01 challenge
func (cm *CertManager) RequestCertificate(aliyunAccessKey, aliyunSecretKey, aliyunRegion string) error {
	// Create a new user key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create a new user
	user := &User{
		Email: cm.Email,
		Key:   privateKey,
	}

	// Create a new ACME client
	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryProduction  // Use production environment
	config.Certificate.KeyType = certcrypto.EC256 // Use EC256 key type

	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create ACME client: %v", err)
	}

	// Create a new Aliyun DNS provider
	provider, err := aliyundns.NewDNSProvider(aliyunAccessKey, aliyunSecretKey, aliyunRegion)
	if err != nil {
		return fmt.Errorf("failed to create DNS provider: %v", err)
	}

	// Set the DNS provider
	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return fmt.Errorf("failed to set DNS provider: %v", err)
	}

	// Register the user
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return fmt.Errorf("failed to register account: %v", err)
	}
	user.Registration = reg

	// Request a certificate
	request := certificate.ObtainRequest{
		Domains: []string{cm.Domain},
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("failed to obtain certificate: %v", err)
	}

	// Save certificate
	if err := os.WriteFile(cm.certPath, certificates.Certificate, 0600); err != nil {
		return fmt.Errorf("failed to save certificate: %v", err)
	}

	// Save private key
	if err := os.WriteFile(cm.keyPath, certificates.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to save private key: %v", err)
	}

	return nil
}

// GetCertificatePaths returns the paths to the certificate and key files
func (cm *CertManager) GetCertificatePaths() (certPath, keyPath string) {
	return cm.certPath, cm.keyPath
}

// LoadCertificate loads the certificate and key from files
func (cm *CertManager) LoadCertificate() (certPEM, keyPEM []byte, err error) {
	certPEM, err = os.ReadFile(cm.certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file: %v", err)
	}

	keyPEM, err = os.ReadFile(cm.keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file: %v", err)
	}

	return certPEM, keyPEM, nil
}
