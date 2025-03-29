package main

import (
	"log"
	"os"

	"github.com/WqyJh/qiniu-ssl/internal/action"
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

			return action.Run(qiniuAccessKey, qiniuSecretKey, aliyunAccessKey, aliyunSecretKey, domain, email, certDir, aliyunRegion, forceHTTPS, http2)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
