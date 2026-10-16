//go:build js && wasm

// Command marzwasm exposes the real marz scanner to the browser demo.
//
//	marz.scan([{path, content}], "IR,CN,RU") -> JSON report string
//	marz.rules() -> JSON knowledge base string
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/mrzroot/marz/internal/kb"
	"github.com/mrzroot/marz/internal/report"
	"github.com/mrzroot/marz/internal/scan"
)

var version = "dev"

func main() {
	k := kb.MustLoad()
	s := scan.New(k, scan.Options{Docs: true})
	api := js.Global().Get("Object").New()
	api.Set("version", version)
	api.Set("scan", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 1 {
			return `{"error":"scan(files, countries)"}`
		}
		files := args[0]
		countries := kb.Countries
		if len(args) > 1 && args[1].Type() == js.TypeString && args[1].String() != "" {
			countries = nil
			for _, c := range strings.Split(args[1].String(), ",") {
				c = strings.ToUpper(strings.TrimSpace(c))
				if kb.CountryNames[c] != "" {
					countries = append(countries, c)
				}
			}
		}
		var hits []scan.Hit
		n := files.Length()
		for i := 0; i < n; i++ {
			f := files.Index(i)
			p := f.Get("path").String()
			if !s.Wanted(p) && !strings.HasSuffix(p, ".md") {
				continue
			}
			hits = append(hits, s.File(p, []byte(f.Get("content").String()))...)
		}
		res := report.Build(k, "browser", n, hits, countries, version)
		var md bytes.Buffer
		report.Markdown(&md, res)
		b, _ := json.Marshal(map[string]any{"result": res, "markdown": md.String()})
		return string(b)
	}))
	api.Set("rules", js.FuncOf(func(this js.Value, args []js.Value) any {
		b, _ := json.Marshal(k)
		return string(b)
	}))
	js.Global().Set("marz", api)
	if cb := js.Global().Get("onMarzReady"); cb.Type() == js.TypeFunction {
		cb.Invoke()
	}
	select {}
}
