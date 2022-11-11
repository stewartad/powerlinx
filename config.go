package powerlinx

type OptionName string

type SiteConfig struct {
	IncludeDrafts bool
}

func NewConfig() *SiteConfig {
	return &SiteConfig{
		IncludeDrafts: false,
	}
}

type SiteOption interface {
	SetSiteOption(*SiteConfig)
}

type includeDrafts struct{}

func (o *includeDrafts) SetSiteOption(c *SiteConfig) {
	c.IncludeDrafts = true
}

func IncludeDrafts() interface {
	SiteOption
} {
	return &includeDrafts{}
}
