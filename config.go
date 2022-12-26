package powerlinx

type OptionName string

type SiteConfig struct {
	Includedrafts bool
	Baseurl       string
	Title         string
	Author        string
	Description   string
}

func NewConfig() *SiteConfig {
	return &SiteConfig{
		Includedrafts: false,
		Baseurl:       "localhost:8080",
	}
}

type SiteOption interface {
	SetSiteOption(*SiteConfig)
}

type includeDrafts struct{}

func (o *includeDrafts) SetSiteOption(c *SiteConfig) {
	c.Includedrafts = true
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
	c.Baseurl = o.url
}

func SetBaseUrl(url string) interface {
	SiteOption
} {
	return &setBaseUrl{
		url: url,
	}
}
