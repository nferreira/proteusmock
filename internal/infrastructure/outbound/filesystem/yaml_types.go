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
	RateLimit *yamlRateLimit `yaml:"rate_limit,omitempty"`
	Latency   *yamlLatency   `yaml:"latency,omitempty"`
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
