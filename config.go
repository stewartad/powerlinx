package powerlinx

type OptionName string

type SiteConfig struct {
	IncludeHidden bool
	BaseUrl       string
}

func NewConfig() *SiteConfig {
	return &SiteConfig{
		IncludeHidden: false,
		BaseUrl:       "localhost:8080",
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

type setBaseUrl struct {
	url string
}

func (o *setBaseUrl) SetSiteOption(c *SiteConfig) {
	c.BaseUrl = o.url
}

func SetBaseUrl(url string) interface {
	SiteOption
} {
	return &setBaseUrl{
		url: url,
	}
}
