package types

type Config struct {
	Database struct {
		ConfigPath string `json:"config_path"`
		DbConfig   struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Name     string `json:"db_name"`
			Driver   string `json:"db_driver"`
		}
	}
	App struct {
		Host struct {
			CertificatePath string `json:"cert_path"`
			KeyPath         string `json:"key_path"`
			Port            int    `json:"port"`
			UseTLS          bool   `json:"use_tls"`
		}
		Cors struct {
			AllowCredentials bool     `json:"allow_credentials"`
			AllowHeaders     []string `json:"allow_headers"`
			AllowOrigins     []string `json:"allow_origins"`
		}
		Limiter struct {
			Expiration               int  `json:"expiration"`
			LimiterSlidingMiddleware bool `json:"limiter_sliding_middleware"`
			Max                      int  `json:"max_requests"`
			SkipSuccessfulRequests   bool `json:"skip_successful_requests"`
		}
		Client struct {
			UserAgent string `json:"user_agent"`
		}
	}
	Images struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
}
