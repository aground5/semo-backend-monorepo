package attribute_engine

import (
	"encoding/json"
	"errors"
	"net"
	"net/mail"
	"strings"

	"semo-server/main/utils"

	"gorm.io/datatypes"
)

type EmailConfig struct {
	BaseConfig      `json:",inline"`
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	VerifyMX        bool     `json:"verify_mx,omitempty"`
	BlockDisposable bool     `json:"block_disposable,omitempty"`
	AllowIDN        bool     `json:"allow_idn,omitempty"`
	MaskEmail       bool     `json:"mask_email,omitempty"`
}

type EmailEngine struct{}

func (ee EmailEngine) ValidateConfig(config datatypes.JSON) ConfigValidationResult {
	var cfg EmailConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ConfigValidationResult{
			IsValid: false,
			Errors:  []string{"invalid email config format"},
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

func (ee EmailEngine) ValidateValue(value string, config datatypes.JSON) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", nil
	}
	addr, err := mail.ParseAddress(v)
	if err != nil {
		return "", errors.New("invalid email format")
	}
	var cfg EmailConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return "", err
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return "", errors.New("invalid email format (missing @)")
	}
	dom := parts[1]
	if len(cfg.AllowedDomains) > 0 {
		allowed := false
		for _, d := range cfg.AllowedDomains {
			if strings.EqualFold(d, dom) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", errors.New("domain not allowed: " + dom)
		}
	}
	if cfg.VerifyMX {
		mxRecords, err := net.LookupMX(dom)
		if err != nil || len(mxRecords) == 0 {
			return "", errors.New("no MX records found for domain: " + dom)
		}
	}
	if cfg.BlockDisposable {
		disposableDomains := []string{"mailinator.com", "10minutemail.com"}
		for _, d := range disposableDomains {
			if strings.EqualFold(d, dom) {
				return "", errors.New("disposable email domains are not allowed")
			}
		}
	}
	if cfg.MaskEmail {
		if len(parts[0]) > 2 {
			v = parts[0][:2] + "****@" + dom
		}
	}
	return v, nil
}

func (ee EmailEngine) GetDisplayInfo(value string, config datatypes.JSON) (DisplayInfo, error) {
	var cfg EmailConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return DisplayInfo{}, err
	}
	color := "#000000"
	if cfg.DisplayColor != "" {
		color = cfg.DisplayColor
	}
	return DisplayInfo{
		DisplayValue: value,
		DisplayColor: color,
	}, nil
}

func (ee EmailEngine) MergeConfig(current, new datatypes.JSON) datatypes.JSON {
	return GenericMergeConfigSimple[EmailConfig](current, new)
}

func (ee EmailEngine) DefaultConfig() datatypes.JSON {
	cfg := EmailConfig{
		BaseConfig: BaseConfig{},
	}
	return StructToJSON(cfg)
}

func (ee EmailEngine) TypeName() string {
	return "email"
}
