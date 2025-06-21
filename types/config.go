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
		Workers struct {
			ImageFetch struct {
				QueryInterval int `json:"query_interval"`
				FetchInterval int `json:"fetch_interval"`
			} `json:"image_fetch"`
		}
	}
	Images struct {
		Path      string `json:"path"`
		Directory string `json:"directory"`
		Type      string `json:"type"`
	}
	Service struct {
		User struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Validate string `json:"validate"`
		}
	}
}
