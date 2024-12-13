package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

func equalslice(l, r reflect.Value) bool {
	if l.Len() != r.Len() {
		return false
	}
	for i := 0; i < l.Len(); i++ {
		if l.Index(i).Kind() == reflect.Slice && r.Index(i).Kind() == reflect.Slice && !equalslice(l.Index(i), r.Index(i)) {
			return false
		}
		if !l.Index(i).Equal(r.Index(i)) {
			return false
		}
	}
	return true
}
func do_diff(l, r any, res map[string]any) {
	lv := reflect.ValueOf(l)
	rv := reflect.ValueOf(r)
	for _, k := range rv.MapKeys() {

		irv := rv.MapIndex(k).Elem()
		slog.Debug("%s:, value kind: %v\n", k.String(), irv.Kind())
		if !lv.MapIndex(k).IsValid() {
			res[k.String()] = irv.Interface()
			continue
		}
		ilv := lv.MapIndex(k).Elem()

		switch irv.Kind() {
		case reflect.Map:
			innermap := make(map[string]any)
			do_diff(ilv.Interface(), irv.Interface(), innermap)
			if len(innermap) != 0 {
				res[k.String()] = innermap
			}
		case reflect.Slice, reflect.Array:
			if !equalslice(ilv, irv) {
				res[k.String()] = irv.Interface()
			}
		case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64,
			reflect.Bool:

			if !ilv.Equal(irv) {
				res[k.String()] = irv.Interface()
				slog.Debug("found diff " + k.String())
			}
		}
	}
}

type Encoder interface {
	Encode(d map[string]any) (string, error)
}
type YAML int

func (i YAML) Encode(d map[string]any) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(int(i))
	err := enc.Encode(d)
	return buf.String(), err
}

type JSON int

func (i JSON) Encode(d map[string]any) (string, error) {
	buf := bytes.NewBuffer(nil)

	enc := json.NewEncoder(buf)
	enc.SetIndent("", strings.Repeat(" ", int(i)))
	err := enc.Encode(d)
	return buf.String(), err
}

type TOML int

func (i TOML) Encode(d map[string]any) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := toml.NewEncoder(buf)
	enc.Indent = strings.Repeat(" ", int(i))
	err := enc.Encode(d)
	return buf.String(), err
}

// diff
func diff(l, r map[string]any) map[string]any {
	result := make(map[string]any)
	do_diff(l, r, result)
	return result
}

func do_merge(l, r any) {
	lv := reflect.ValueOf(l)
	rv := reflect.ValueOf(r)
	for _, k := range rv.MapKeys() {

		irv := rv.MapIndex(k).Elem()

		log := slog.With(k.String(), irv.Kind())
		if !lv.MapIndex(k).IsValid() {
			log.Info("letf not valied ")
			lv.SetMapIndex(k, rv.MapIndex(k))
			continue
		}
		ilv := lv.MapIndex(k).Elem()

		switch irv.Kind() {
		case reflect.Map:
			log.Debug("right map ", "left kind", ilv.Kind())
			if ilv.Kind() != reflect.Map {
				lv.SetMapIndex(k, rv.MapIndex(k))
			} else {
				do_merge(ilv.Interface(), irv.Interface())
			}

		default:
			log.Debug("set  value ")
			lv.SetMapIndex(k, rv.MapIndex(k))
		}
	}
}

// marge r to l
func merge(l, r map[string]any) map[string]any {
	if l == nil {
		l = make(map[string]any)
	}
	do_merge(l, r)
	return l
}

func main() {
	//slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	f := flag.String("f", "yaml", "input format:yaml/json/toml")
	t := flag.String("t", "yaml", "output format:yaml/json/toml")
	indent := flag.Int("i", 2, "output indent")
	op := flag.String("o", "diff", "operator:diff/merge")
	l := flag.String("l", "", "left file")
	r := flag.String("r", "", "right file")
	flag.Parse()
	lc, err := os.ReadFile(*l)
	if err != nil {
		fmt.Printf("read left file failed %v\n", err)
		os.Exit(1)
		return
	}
	rc, err := os.ReadFile(*r)
	if err != nil {
		fmt.Printf("read right file failed %v\n", err)
		os.Exit(1)
		return
	}
	var decoder func(in []byte, out interface{}) (err error)
	switch *f {
	case "yaml":
		decoder = yaml.Unmarshal
	case "json":
		decoder = json.Unmarshal
	case "toml":
		decoder = toml.Unmarshal
	default:
		fmt.Printf("unkonw input format\n")
		os.Exit(1)
	}
	var encoder Encoder
	switch *t {
	case "yaml":
		encoder = YAML(*indent)
	case "json":
		encoder = JSON(*indent)
	case "toml":
		encoder = TOML(*indent)
	default:
		fmt.Printf("unkonw output format\n")
		os.Exit(1)
	}
	var opf func(l, r map[string]any) map[string]any
	switch *op {
	case "diff":
		opf = diff
	case "merge":
		opf = merge
	default:
		fmt.Printf("unkonw op\n")
		os.Exit(1)
	}

	var lm, rm map[string]any
	err = decoder(lc, &lm)
	slog.Debug(fmt.Sprintf("%+v]", lm))
	if err != nil {
		fmt.Printf("decode left file failed %v\n", err)
		os.Exit(1)
	}
	err = decoder(rc, &rm)
	slog.Debug(fmt.Sprintf("%+v]", rm))
	if err != nil {
		fmt.Printf("decode right file failed %v\n", err)
		os.Exit(1)
	}
	res := opf(lm, rm)

	out, err := encoder.Encode(res)
	if err != nil {
		fmt.Printf("encode failed %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}
