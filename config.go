package powerlinx

type OptionName string

type SiteConfig struct {
	IncludeHidden bool
}

func NewConfig() *SiteConfig {
	return &SiteConfig{
		IncludeHidden: false,
	}
}

type SiteOption interface {
	SetSiteOption(*SiteConfig)
}

type includeDrafts struct{}

func (o *includeDrafts) SetSiteOption(c *SiteConfig) {
	c.IncludeHidden = true
}

func IncludeDrafts() interface {
	SiteOption
} {
	return &includeDrafts{}
}
