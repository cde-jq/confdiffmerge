package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

func kind(v reflect.Value) reflect.Kind {
	r := v.Kind()
	if r == reflect.Interface {
		return v.Elem().Kind()
	}
	return r
}
func intf(v reflect.Value) any {
	r := v.Kind()
	if r == reflect.Interface {
		return v.Elem().Interface()
	}
	return v.Interface()
}
func equalslice(l, r reflect.Value) bool {
	if l.Len() != r.Len() {
		return false
	}
	for i := 0; i < l.Len(); i++ {

		if l.Index(i).Kind() == reflect.Slice && r.Index(i).Kind() == reflect.Slice && !equalslice(l.Index(i), r.Index(i)) {
			return false
		}

		if kind(r.Index(i)) == reflect.Map || kind(l.Index(i)) == reflect.Map {
			if kind(r.Index(i)) != kind(l.Index(i)) {
				return false
			}
			tmp := make(map[string]any)
			slog.Debug("", "rkind", kind(r.Index(i)), "lkind", kind(l.Index(i)))
			do_diff(intf(l.Index(i)), intf(r.Index(i)), tmp)
			return len(tmp) == 0
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
		slog.Debug("process key", k.String(), irv.Kind())
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

		if !lv.MapIndex(k).IsValid() {

			lv.SetMapIndex(k, rv.MapIndex(k))
			continue
		}
		ilv := lv.MapIndex(k).Elem()

		switch irv.Kind() {
		case reflect.Map:

			if ilv.Kind() != reflect.Map {
				lv.SetMapIndex(k, rv.MapIndex(k))
			} else {
				do_merge(ilv.Interface(), irv.Interface())
			}

		default:

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

var usage string = `Usage of %s:

diff or merge right conf to left conf
diff op will find changed and added options

Parameters:
`

func main() {

	frdefault := "same to left input format(-f)"

	flag.Usage = func() {
		fmt.Printf(usage, path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	f := flag.String("f", "yaml", "left input format:yaml/json/toml")
	fr := flag.String("fr", frdefault, "right input format:yaml/json/toml")
	t := flag.String("t", frdefault, "output format:yaml/json/toml")
	indent := flag.Int("i", 2, "output indent")
	op := flag.String("o", "diff", "operator:diff/merge")
	l := flag.String("l", "", "left conf file")
	r := flag.String("r", "", "right conf file")
	p := flag.String("p", "", "prefix in each line")
	debug := flag.Bool("d", false, "debug log")
	flag.Parse()
	if *fr == frdefault {
		fr = f
	}
	if *t == frdefault {
		t = f
	}
	loglevel := slog.LevelInfo
	if *debug {
		loglevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loglevel})))
	if *l == "" || *r == "" {
		fmt.Println("Err: must specify left and right file.")
		flag.Usage()
		os.Exit(1)

	}
	lc, err := os.ReadFile(*l)
	if err != nil {
		fmt.Printf("read left file failed %v\n", err)
		os.Exit(1)

	}

	rc, err := os.ReadFile(*r)
	if err != nil {
		fmt.Printf("read right file failed %v\n", err)
		os.Exit(1)

	}
	type Decoder func(in []byte, out interface{}) (err error)

	getdecoder := func(format string) Decoder {
		switch format {
		case "yaml":
			return yaml.Unmarshal
		case "json":
			return json.Unmarshal
		case "toml":
			return toml.Unmarshal
		default:
			fmt.Printf("unkonw format\n")
			os.Exit(1)
		}
		panic("unreachable")
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
	err = getdecoder(*f)(lc, &lm)
	if err != nil {
		fmt.Printf("decode left file failed %v\n", err)
		os.Exit(1)
	}
	err = getdecoder(*fr)(rc, &rm)
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

	lines := strings.Split(out, "\n")
	for c, l := range lines {
		if len(l) != 0 || len(lines)-1 != c {
			fmt.Print(*p)
			fmt.Println(l)
		}
	}

}
