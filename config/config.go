package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DefaultConfigPath = "config/.env"
	ServiceName       = "ssi-service"
	DefaultServiceEndpoint = "http://localhost:8080"

	// Environment variable names
	EnvVersion                   = "SSI_SERVICE_VERSION"
	EnvDescription              = "SSI_SERVICE_DESCRIPTION"
	EnvServerAPIHost            = "SSI_SERVICE_SERVER_API_HOST"
	EnvServerDebugHost          = "SSI_SERVICE_SERVER_DEBUG_HOST"
	EnvServerJagerHost          = "SSI_SERVICE_SERVER_JAGER_HOST"
	EnvServerJagerEnabled       = "SSI_SERVICE_SERVER_JAGER_ENABLED"
	EnvServerReadTimeout        = "SSI_SERVICE_SERVER_READ_TIMEOUT"
	EnvServerWriteTimeout       = "SSI_SERVICE_SERVER_WRITE_TIMEOUT"
	EnvServerShutdownTimeout    = "SSI_SERVICE_SERVER_SHUTDOWN_TIMEOUT"
	EnvServerLogLocation        = "SSI_SERVICE_SERVER_LOG_LOCATION"
	EnvServerLogLevel           = "SSI_SERVICE_SERVER_LOG_LEVEL"
	EnvServerEnableSchemaCaching = "SSI_SERVICE_SERVER_ENABLE_SCHEMA_CACHING"
	EnvServerEnableAllowAllCORS = "SSI_SERVICE_SERVER_ENABLE_ALLOW_ALL_CORS"
	EnvServicesEndpoint         = "SSI_SERVICE_SERVICES_ENDPOINT"
	EnvServicesStorage          = "SSI_SERVICE_SERVICES_STORAGE"
	EnvServicesStorageAddress   = "SSI_SERVICE_SERVICES_STORAGE_ADDRESS"
	EnvServicesStoragePassword  = "SSI_SERVICE_SERVICES_STORAGE_PASSWORD"
	EnvServicesKeystoreName     = "SSI_SERVICE_SERVICES_KEYSTORE_NAME"
	EnvServicesKeystorePassword = "SSI_SERVICE_SERVICES_KEYSTORE_PASSWORD"
	EnvServicesDIDName          = "SSI_SERVICE_SERVICES_DID_NAME"
	EnvServicesDIDMethods       = "SSI_SERVICE_SERVICES_DID_METHODS"
	EnvServicesDIDResolutionMethods = "SSI_SERVICE_SERVICES_DID_RESOLUTION_METHODS"
	EnvServicesSchemaName       = "SSI_SERVICE_SERVICES_SCHEMA_NAME"
	EnvServicesIssuingName      = "SSI_SERVICE_SERVICES_ISSUING_NAME"
	EnvServicesCredentialName   = "SSI_SERVICE_SERVICES_CREDENTIAL_NAME"
	EnvServicesManifestName     = "SSI_SERVICE_SERVICES_MANIFEST_NAME"
	EnvServicesPresentationName = "SSI_SERVICE_SERVICES_PRESENTATION_NAME"
)

type EnvironmentVariable string

type SSIServiceConfig struct {
	conf.Version
	Server   ServerConfig   `conf:"server"`
	Services ServicesConfig `conf:"services"`
}

// ServerConfig represents configurable properties for the HTTP server
type ServerConfig struct {
	APIHost             string        `conf:"default:0.0.0.0:3000"`
	DebugHost           string        `conf:"default:0.0.0.0:4000"`
	JagerHost           string        `conf:"http://jaeger:14268/api/traces"`
	JagerEnabled        bool          `conf:"default:false"`
	ReadTimeout         time.Duration `conf:"default:5s"`
	WriteTimeout        time.Duration `conf:"default:5s"`
	ShutdownTimeout     time.Duration `conf:"default:5s"`
	LogLocation         string        `conf:"default:log"`
	LogLevel            string        `conf:"default:debug"`
	EnableSchemaCaching bool          `conf:"default:true"`
	EnableAllowAllCORS  bool          `conf:"default:false"`
}

type IssuingServiceConfig struct {
	*BaseServiceConfig
}

func (s *IssuingServiceConfig) IsEmpty() bool {
	if s == nil {
		return true
	}
	return reflect.DeepEqual(s, &IssuingServiceConfig{})
}

// ServicesConfig represents configurable properties for the components of the SSI Service
type ServicesConfig struct {
	StorageProvider string      `conf:"default:bolt"`
	StorageOption   interface{} `conf:"default:{}"`
	ServiceEndpoint string      `conf:"default:http://localhost:8080"`

	// Embed all service-specific configs here
	KeyStoreConfig       KeyStoreServiceConfig     `conf:"keystore,omitempty"`
	DIDConfig            DIDServiceConfig          `conf:"did,omitempty"`
	IssuingServiceConfig IssuingServiceConfig      `conf:"issuing,omitempty"`
	SchemaConfig         SchemaServiceConfig       `conf:"schema,omitempty"`
	CredentialConfig     CredentialServiceConfig   `conf:"credential,omitempty"`
	ManifestConfig       ManifestServiceConfig     `conf:"manifest,omitempty"`
	PresentationConfig   PresentationServiceConfig `conf:"presentation,omitempty"`
	WebhookConfig        WebhookServiceConfig      `conf:"webhook,omitempty"`
}

// BaseServiceConfig represents configurable properties for a specific component of the SSI Service
type BaseServiceConfig struct {
	Name            string `conf:"name"`
	ServiceEndpoint string `conf:"service_endpoint"`
}

type KeyStoreServiceConfig struct {
	*BaseServiceConfig
	ServiceKeyPassword string `conf:"default:default-password"`
}

func (k *KeyStoreServiceConfig) IsEmpty() bool {
	if k == nil {
		return true
	}
	return reflect.DeepEqual(k, &KeyStoreServiceConfig{})
}

type DIDServiceConfig struct {
	*BaseServiceConfig
	Methods           []string `conf:"default:key,web"`
	ResolutionMethods []string `conf:"default:key,peer,web,pkh"`
}

func (d *DIDServiceConfig) IsEmpty() bool {
	if d == nil {
		return true
	}
	return reflect.DeepEqual(d, &DIDServiceConfig{})
}

type SchemaServiceConfig struct {
	*BaseServiceConfig
}

func (s *SchemaServiceConfig) IsEmpty() bool {
	if s == nil {
		return true
	}
	return reflect.DeepEqual(s, &SchemaServiceConfig{})
}

type CredentialServiceConfig struct {
	*BaseServiceConfig
}

func (c *CredentialServiceConfig) IsEmpty() bool {
	if c == nil {
		return true
	}
	return reflect.DeepEqual(c, &CredentialServiceConfig{})
}

type ManifestServiceConfig struct {
	*BaseServiceConfig
}

func (m *ManifestServiceConfig) IsEmpty() bool {
	if m == nil {
		return true
	}
	return reflect.DeepEqual(m, &ManifestServiceConfig{})
}

type PresentationServiceConfig struct {
	*BaseServiceConfig
}

func (p *PresentationServiceConfig) IsEmpty() bool {
	if p == nil {
		return true
	}
	return reflect.DeepEqual(p, &PresentationServiceConfig{})
}

type WebhookServiceConfig struct {
	*BaseServiceConfig
}

func (p *WebhookServiceConfig) IsEmpty() bool {
	if p == nil {
		return true
	}
	return reflect.DeepEqual(p, &WebhookServiceConfig{})
}

// LoadConfig loads configuration from environment variables
func LoadConfig(path string) (*SSIServiceConfig, error) {
	// Load .env file if provided
	if path != "" {
		if filepath.Ext(path) != ".env" {
			return nil, fmt.Errorf("path<%s> must be a .env file", path)
		}
		if err := godotenv.Load(path); err != nil {
			if os.IsNotExist(err) {
				logrus.Info("no .env file found, proceeding with environment variables...")
			} else {
				return nil, errors.Wrap(err, "loading .env file")
			}
		}
	}

	// Create the config object
	var config SSIServiceConfig

	// Apply defaults
	if err := conf.Parse(os.Args[1:], ServiceName, &config); err != nil {
		switch {
		case errors.Is(err, conf.ErrHelpWanted):
			usage, err := conf.Usage(ServiceName, &config)
			if err != nil {
				return nil, errors.Wrap(err, "parsing config")
			}
			fmt.Println(usage)
			return nil, nil
		case errors.Is(err, conf.ErrVersionWanted):
			version, err := conf.VersionString(ServiceName, &config)
			if err != nil {
				return nil, errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil, nil
		}
		return nil, errors.Wrap(err, "parsing config")
	}

	// Load default services config if no specific overrides are provided
	loadDefaultServicesConfig(&config)

	// Apply environment variables
	if err := applyEnvVariables(&config); err != nil {
		return nil, errors.Wrap(err, "applying environment variables")
	}

	return &config, nil
}

func loadDefaultServicesConfig(config *SSIServiceConfig) {
	servicesConfig := ServicesConfig{
		StorageProvider: "bolt",
		ServiceEndpoint: DefaultServiceEndpoint,
		KeyStoreConfig: KeyStoreServiceConfig{
			BaseServiceConfig:  &BaseServiceConfig{Name: "keystore"},
			ServiceKeyPassword: "default-password",
		},
		DIDConfig: DIDServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "did"},
			Methods:           []string{"key", "web"},
			ResolutionMethods: []string{"key", "peer", "web", "pkh"},
		},
		SchemaConfig: SchemaServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "schema"},
		},
		CredentialConfig: CredentialServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "credential", ServiceEndpoint: DefaultServiceEndpoint},
		},
		ManifestConfig: ManifestServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "manifest"},
		},
		PresentationConfig: PresentationServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "presentation"},
		},
		IssuingServiceConfig: IssuingServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "issuing"},
		},
		WebhookConfig: WebhookServiceConfig{
			BaseServiceConfig: &BaseServiceConfig{Name: "webhook"},
		},
	}

	config.Services = servicesConfig
}

func applyEnvVariables(config *SSIServiceConfig) error {
	// Version and Description
	if v := os.Getenv(EnvVersion); v != "" {
		config.Version.SVN = v
	}
	if d := os.Getenv(EnvDescription); d != "" {
		config.Version.Desc = d
	}

	// Server Configuration
	if v := os.Getenv(EnvServerAPIHost); v != "" {
		config.Server.APIHost = v
	}
	if v := os.Getenv(EnvServerDebugHost); v != "" {
		config.Server.DebugHost = v
	}
	if v := os.Getenv(EnvServerJagerHost); v != "" {
		config.Server.JagerHost = v
	}
	if v := os.Getenv(EnvServerJagerEnabled); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			config.Server.JagerEnabled = b
		}
	}
	if v := os.Getenv(EnvServerReadTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv(EnvServerWriteTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv(EnvServerShutdownTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.Server.ShutdownTimeout = d
		}
	}
	if v := os.Getenv(EnvServerLogLocation); v != "" {
		config.Server.LogLocation = v
	}
	if v := os.Getenv(EnvServerLogLevel); v != "" {
		config.Server.LogLevel = v
	}
	if v := os.Getenv(EnvServerEnableSchemaCaching); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			config.Server.EnableSchemaCaching = b
		}
	}
	if v := os.Getenv(EnvServerEnableAllowAllCORS); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			config.Server.EnableAllowAllCORS = b
		}
	}

	// Services Configuration
	if v := os.Getenv(EnvServicesEndpoint); v != "" {
		config.Services.ServiceEndpoint = v
	}
	if v := os.Getenv(EnvServicesStorage); v != "" {
		config.Services.StorageProvider = v
	}

	// Storage Options
	storageOption := make(map[string]interface{})
	if config.Services.StorageOption != nil {
		if m, ok := config.Services.StorageOption.(map[string]interface{}); ok {
			storageOption = m
		}
	}
	if v := os.Getenv(EnvServicesStorageAddress); v != "" {
		storageOption["address"] = v
	}
	if v := os.Getenv(EnvServicesStoragePassword); v != "" {
		storageOption["password"] = v
	}
	config.Services.StorageOption = storageOption

	// Service-specific Configurations
	if v := os.Getenv(EnvServicesKeystoreName); v != "" {
		config.Services.KeyStoreConfig.Name = v
	}
	if v := os.Getenv(EnvServicesKeystorePassword); v != "" {
		config.Services.KeyStoreConfig.ServiceKeyPassword = v
	}
	if v := os.Getenv(EnvServicesDIDName); v != "" {
		config.Services.DIDConfig.Name = v
	}
	if v := os.Getenv(EnvServicesDIDMethods); v != "" {
		config.Services.DIDConfig.Methods = strings.Split(v, ",")
	}
	if v := os.Getenv(EnvServicesDIDResolutionMethods); v != "" {
		config.Services.DIDConfig.ResolutionMethods = strings.Split(v, ",")
	}
	if v := os.Getenv(EnvServicesSchemaName); v != "" {
		config.Services.SchemaConfig.Name = v
	}
	if v := os.Getenv(EnvServicesIssuingName); v != "" {
		config.Services.IssuingServiceConfig.Name = v
	}
	if v := os.Getenv(EnvServicesCredentialName); v != "" {
		config.Services.CredentialConfig.Name = v
	}
	if v := os.Getenv(EnvServicesManifestName); v != "" {
		config.Services.ManifestConfig.Name = v
	}
	if v := os.Getenv(EnvServicesPresentationName); v != "" {
		config.Services.PresentationConfig.Name = v
	}

	return nil
}