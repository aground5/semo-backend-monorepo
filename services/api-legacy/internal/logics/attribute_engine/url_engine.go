package attribute_engine

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"semo-server/main/utils"

	"gorm.io/datatypes"
)

type URLConfig struct {
	BaseConfig       `json:",inline"`
	RequireScheme    bool     `json:"require_scheme,omitempty"`
	AllowedDomains   []string `json:"allowed_domains,omitempty"`
	AllowedProtocols []string `json:"allowed_protocols,omitempty"`
	Relative         bool     `json:"relative,omitempty"`
	Normalize        bool     `json:"normalize,omitempty"`
}

type URLEngine struct{}

func (ue URLEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg URLConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid url config format"},
		}
	}
	var errs []string
	if cfg.DisplayColor != "" {
		if err := utils.ValidateColorString(cfg.DisplayColor); err != nil {
			errs = append(errs, "display_color: "+err.Error())
		}
	}
	return ConfigValidationResult{
		IsValid: len(errs) == 0,
		Errors:  errs,
		Fixed:   StructToJSON(cfg),
	}
}

func (ue URLEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	var cfg URLConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	u, err := url.Parse(v)
	if err != nil {
		return "", errors.New("invalid URL format")
	}
	if cfg.RequireScheme && (u.Scheme == "" || u.Host == "") {
		return "", errors.New("scheme/host is required but missing")
	}
	if len(cfg.AllowedProtocols) > 0 && u.Scheme != "" {
		allowed := false
		for _, proto := range cfg.AllowedProtocols {
			if strings.EqualFold(u.Scheme, proto) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", errors.New("protocol not allowed: " + u.Scheme)
		}
	}
	if len(cfg.AllowedDomains) > 0 && u.Host != "" {
		host := strings.ToLower(u.Host)
		allowed := false
		for _, d := range cfg.AllowedDomains {
			if strings.EqualFold(host, d) || strings.HasSuffix(host, "."+strings.ToLower(d)) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", errors.New("domain not allowed: " + u.Host)
		}
	}
	if cfg.Normalize {
		u.Scheme = strings.ToLower(u.Scheme)
		u.Host = strings.ToLower(u.Host)
		v = u.String()
	}
	return v, nil
}

func (ue URLEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg URLConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	color := "#0000EE"
	if cfg.DisplayColor != "" {
		color = cfg.DisplayColor
	}
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: color,
	}, nil
}

func (ue URLEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[URLConfig](current, new)
}

func (ue URLEngine) DefaultConfig() datatypes.JSON {
	cfg := URLConfig{
		BaseConfig: BaseConfig{},
	}
	return StructToJSON(cfg)
}

func (ue URLEngine) TypeName() string {
	return "url"
}
