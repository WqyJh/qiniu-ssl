package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/WqyJh/qiniu-ssl/internal/action"
	"github.com/WqyJh/qiniu-ssl/internal/qiniuapi"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "qiniu-ssl",
		Usage: "Apply for Let's Encrypt SSL certificates using Aliyun DNS challenge and upload to Qiniu CDN",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "qiniu-access-key",
				Aliases: []string{"qak"},
				Usage:   "Qiniu access key",
				EnvVars: []string{"QINIU_ACCESS_KEY"},
			},
			&cli.StringFlag{
				Name:    "qiniu-secret-key",
				Aliases: []string{"qsk"},
				Usage:   "Qiniu secret key",
				EnvVars: []string{"QINIU_SECRET_KEY"},
			},
			&cli.StringFlag{
				Name:    "aliyun-access-key",
				Aliases: []string{"aak"},
				Usage:   "Aliyun access key for DNS challenge",
				EnvVars: []string{"ALIYUN_ACCESS_KEY"},
			},
			&cli.StringFlag{
				Name:    "aliyun-secret-key",
				Aliases: []string{"ask"},
				Usage:   "Aliyun secret key for DNS challenge",
				EnvVars: []string{"ALIYUN_SECRET_KEY"},
			},
			&cli.StringFlag{
				Name:    "aliyun-region",
				Aliases: []string{"ar"},
				Usage:   "Aliyun region for DNS API",
				Value:   "cn-hangzhou",
				EnvVars: []string{"ALIYUN_REGION"},
			},
			&cli.StringFlag{
				Name:    "domain",
				Aliases: []string{"d"},
				Usage:   "Domain name for the certificate",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "email",
				Aliases: []string{"e"},
				Usage:   "Email address for Let's Encrypt",
				Value:   "",
			},
			&cli.StringFlag{
				Name:    "cert-dir",
				Aliases: []string{"c"},
				Usage:   "Directory to store certificates",
				Value:   "certs",
			},
			&cli.BoolFlag{
				Name:    "force-https",
				Aliases: []string{"f"},
				Usage:   "Force HTTPS for the domain",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "http2",
				Aliases: []string{"h2"},
				Usage:   "Enable HTTP/2 for the domain",
				Value:   true,
			},
			&cli.IntFlag{
				Name:    "check-interval",
				Aliases: []string{"i"},
				Usage:   "Interval in days between certificate expiry checks",
				Value:   7,
			},
			&cli.IntFlag{
				Name:    "threshold",
				Aliases: []string{"t"},
				Usage:   "Number of days before expiry to trigger renewal",
				Value:   30,
			},
			&cli.BoolFlag{
				Name:  "daemon",
				Usage: "Run as a daemon, checking periodically",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "log-file",
				Usage: "Log file path (if not specified, logs to stdout)",
				Value: "",
			},
			&cli.StringSliceFlag{
				Name:    "domains-file",
				Aliases: []string{"df"},
				Usage:   "Path to file containing list of domains to check (one domain per line)",
				Value:   nil,
			},
		},
		Action: func(c *cli.Context) error {
			qiniuAccessKey := c.String("qiniu-access-key")
			qiniuSecretKey := c.String("qiniu-secret-key")
			aliyunAccessKey := c.String("aliyun-access-key")
			aliyunSecretKey := c.String("aliyun-secret-key")
			aliyunRegion := c.String("aliyun-region")
			domain := c.String("domain")
			email := c.String("email")
			certDir := c.String("cert-dir")
			forceHTTPS := c.Bool("force-https")
			http2 := c.Bool("http2")
			checkInterval := c.Int("check-interval")
			threshold := c.Int("threshold")
			daemon := c.Bool("daemon")
			logFile := c.String("log-file")
			domainsFiles := c.StringSlice("domains-file")

			// Configure logging
			if logFile != "" {
				f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("failed to open log file: %v", err)
				}
				defer f.Close()
				log.SetOutput(f)
			}

			// Get domains from file if specified
			domains := []string{}
			if domain != "" {
				domains = append(domains, domain)
			}

			for _, domainsFile := range domainsFiles {
				content, err := os.ReadFile(domainsFile)
				if err != nil {
					return fmt.Errorf("failed to read domains file %s: %v", domainsFile, err)
				}

				lines := splitLines(string(content))
				for _, line := range lines {
					if line != "" {
						domains = append(domains, line)
					}
				}
			}

			if len(domains) == 0 {
				return fmt.Errorf("no domains specified, use --domain or --domains-file")
			}

			// Validate required parameters
			if qiniuAccessKey == "" || qiniuSecretKey == "" {
				return fmt.Errorf("qiniu access key and secret key are required")
			}

			if aliyunAccessKey == "" || aliyunSecretKey == "" {
				return fmt.Errorf("aliyun access key and secret key are required")
			}

			// Ensure certificate directory exists
			if err := os.MkdirAll(certDir, 0700); err != nil {
				return fmt.Errorf("failed to create certificate directory: %v", err)
			}

			if checkInterval <= 0 {
				return fmt.Errorf("check interval must be greater than 0")
			}

			if threshold <= 0 {
				return fmt.Errorf("threshold must be greater than 0")
			}

			if checkInterval >= threshold {
				return fmt.Errorf("check interval must be less than threshold")
			}

			// Create a channel to handle termination signals
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Create Qiniu client for API operations
			qiniuClient, err := qiniuapi.NewQiniuClient(qiniuAccessKey, qiniuSecretKey)
			if err != nil {
				return fmt.Errorf("failed to create Qiniu client: %v", err)
			}

			// Function to check and renew certificates for all domains
			checkAndRenewAll := func() error {
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				log.Printf("[%s] Checking certificates for %d domains", timestamp, len(domains))

				for _, domainName := range domains {
					log.Printf("Processing domain: %s", domainName)
					// Check certificate directly from Qiniu API
					needsRenewal, certInfo, err := qiniuClient.CheckCertificateFromQiniu(domainName, threshold)
					if err != nil {
						// If there's an error (like no HTTPS or certificate), assume we need to create one
						log.Printf("Error checking certificate for %s from Qiniu: %v", domainName, err)
						log.Printf("Will attempt to request new certificate for %s", domainName)
						needsRenewal = true
					} else if needsRenewal {
						expiresAt := time.Unix(certInfo.NotAfter, 0)
						daysLeft := int(time.Until(expiresAt).Hours() / 24)
						log.Printf("Certificate for %s is expiring on %s (in %d days), renewing...",
							domainName, expiresAt.Format("2006-01-02"), daysLeft)
					} else {
						expiresAt := time.Unix(certInfo.NotAfter, 0)
						daysLeft := int(time.Until(expiresAt).Hours() / 24)
						log.Printf("Certificate for %s is valid until %s (%d days), no renewal needed",
							domainName, expiresAt.Format("2006-01-02"), daysLeft)
						continue
					}

					// Request new certificate and update it on Qiniu
					log.Printf("Requesting and uploading new certificate for %s...", domainName)
					if err := action.Run(qiniuAccessKey, qiniuSecretKey, aliyunAccessKey, aliyunSecretKey,
						domainName, email, certDir, aliyunRegion, forceHTTPS, http2); err != nil {
						log.Printf("Failed to renew certificate for %s: %v", domainName, err)
						continue
					}

					log.Printf("Certificate for %s has been renewed successfully", domainName)
				}

				return nil
			}

			// Run once immediately
			if err := checkAndRenewAll(); err != nil {
				return fmt.Errorf("error during certificate check: %w", err)
			}

			// If daemon mode is enabled, keep checking at the specified interval
			if daemon {
				log.Printf("Running in daemon mode, checking certificates every %d days", checkInterval)

				// Calculate duration in hours
				interval := time.Duration(checkInterval*24) * time.Hour

				ticker := time.NewTicker(interval)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						if err := checkAndRenewAll(); err != nil {
							log.Printf("Error during certificate check: %v", err)
						}
					case sig := <-sigChan:
						log.Printf("Received signal %v, shutting down...", sig)
						return nil
					}
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines
}
