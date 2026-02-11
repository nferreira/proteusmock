package filesystem

// yamlScenario is the YAML deserialization target for scenario files.
type yamlScenario struct {
	ID       string       `yaml:"id"`
	Name     string       `yaml:"name"`
	Priority int          `yaml:"priority"`
	When     yamlWhen     `yaml:"when"`
	Response yamlResponse `yaml:"response"`
	Policy   *yamlPolicy  `yaml:"policy,omitempty"`
}

type yamlWhen struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    *yamlBody         `yaml:"body,omitempty"`
}

type yamlBody struct {
	ContentType string          `yaml:"content_type,omitempty"`
	Conditions  []yamlCondition `yaml:"conditions,omitempty"`
	All         []yamlBody      `yaml:"all,omitempty"`
	Any         []yamlBody      `yaml:"any,omitempty"`
	Not         *yamlBody       `yaml:"not,omitempty"`
}

type yamlCondition struct {
	Extractor string `yaml:"extractor"`
	Matcher   string `yaml:"matcher"`
}

type yamlResponse struct {
	Status      int               `yaml:"status"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	Body        string            `yaml:"body,omitempty"`
	BodyFile    string            `yaml:"body_file,omitempty"`
	ContentType string            `yaml:"content_type,omitempty"`
	Engine      string            `yaml:"engine,omitempty"`
}

type yamlPolicy struct {
	RateLimit  *yamlRateLimit  `yaml:"rate_limit,omitempty"`
	Latency    *yamlLatency    `yaml:"latency,omitempty"`
	Pagination *yamlPagination `yaml:"pagination,omitempty"`
}

type yamlRateLimit struct {
	Rate  float64 `yaml:"rate"`
	Burst int     `yaml:"burst"`
	Key   string  `yaml:"key,omitempty"`
}

type yamlLatency struct {
	FixedMs  int `yaml:"fixed_ms,omitempty"`
	JitterMs int `yaml:"jitter_ms,omitempty"`
}

type yamlPagination struct {
	Style       string                  `yaml:"style,omitempty"`
	PageParam   string                  `yaml:"page_param,omitempty"`
	SizeParam   string                  `yaml:"size_param,omitempty"`
	OffsetParam string                  `yaml:"offset_param,omitempty"`
	LimitParam  string                  `yaml:"limit_param,omitempty"`
	DefaultSize int                     `yaml:"default_size,omitempty"`
	MaxSize     int                     `yaml:"max_size,omitempty"`
	DataPath    string                  `yaml:"data_path,omitempty"`
	Envelope    *yamlPaginationEnvelope `yaml:"envelope,omitempty"`
}

type yamlPaginationEnvelope struct {
	DataField        string `yaml:"data_field,omitempty"`
	PageField        string `yaml:"page_field,omitempty"`
	SizeField        string `yaml:"size_field,omitempty"`
	TotalItemsField  string `yaml:"total_items_field,omitempty"`
	TotalPagesField  string `yaml:"total_pages_field,omitempty"`
	HasNextField     string `yaml:"has_next_field,omitempty"`
	HasPreviousField string `yaml:"has_previous_field,omitempty"`
}
