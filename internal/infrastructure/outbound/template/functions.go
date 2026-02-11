package template

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

func buildExprEnv(ctx match.RenderContext) exprEnv {
	return exprEnv{
		PathParam: func(name string) string {
			return ctx.PathParams[name]
		},
		QueryParam: func(name string) string {
			return ctx.QueryParams[name]
		},
		Header: func(name string) string {
			// Case-insensitive header lookup.
			for k, v := range ctx.Headers {
				if strings.EqualFold(k, name) {
					return v
				}
			}
			return ""
		},
		Body: func() string {
			return string(ctx.Body)
		},
		Now: func() string {
			return ctx.Now
		},
		NowFormat: func(layout string) string {
			t, err := time.Parse(time.RFC3339, ctx.Now)
			if err != nil {
				return ctx.Now
			}
			return t.Format(layout)
		},
		UUID: func() string {
			return generateUUID()
		},
		RandomInt: func(min, max int) int {
			if min >= max {
				return min
			}
			return min + randIntN(max-min+1)
		},
		Seq: func(start, end int) []int {
			return seqInts(start, end)
		},
		ToJSON: func(v any) string {
			return toJSONString(v)
		},
		JsonPath: func(expression string) string {
			return extractJSONPath(ctx.Body, expression)
		},
	}
}

func seqInts(start, end int) []int {
	if end < start {
		return nil
	}
	s := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		s = append(s, i)
	}
	return s
}

func randIntN(n int) int {
	return rand.IntN(n)
}

func toJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func extractJSONPath(body []byte, expression string) string {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}
	result, err := jsonpath.Get(expression, data)
	if err != nil {
		return ""
	}
	switch v := result.(type) {
	case string:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

func generateUUID() string {
	var uuid [16]byte
	for i := range uuid {
		uuid[i] = byte(rand.IntN(256))
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// Pongo2 helper functions — used by the Jinja2 adapter.

func pongo2PathParam(ctx match.RenderContext) func(string) string {
	return func(name string) string { return ctx.PathParams[name] }
}

func pongo2QueryParam(ctx match.RenderContext) func(string) string {
	return func(name string) string { return ctx.QueryParams[name] }
}

func pongo2Header(ctx match.RenderContext) func(string) string {
	return func(name string) string {
		for k, v := range ctx.Headers {
			if strings.EqualFold(k, name) {
				return v
			}
		}
		return ""
	}
}
