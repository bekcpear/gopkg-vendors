// Copyright (C) 2023 Michael J. Fromberger. All Rights Reserved.

// Package flax implements a helper for attaching flags to the fields of
// struct values.
//
// # Overview
//
// Typical use is to define one or more structs carrying groups of related
// flags, identified by field tags. For example, here is a struct with three
// flags:
//
//	var flags struct {
//	   Input  string `flag:"input,Input file name (required)"`
//	   DryRun bool   `flag:"dry-run,Dry run, do not make any changes"`
//	   Count  int    `flag:"count,default=1,Number of iteration"`
//	}
//
// The [Check] function identifies tagged fields that can then be bound to a
// standard [flag.FlagSet] for flag parsing. For example:
//
//	fs, err := flax.Check(&flags)
//	...
//	fs.Bind(flag.CommandLine)
//	...
//	flag.Parse()
//
// The [Check] function reports an error if its argument does not contain any
// valid flag targets. The [Check] documentation details the field tag format
// and flag rules.
//
// For the common case of binding flags at program initialization, the
// [MustBind] and [MustBindAll] functions combine these two steps, with a panic
// in case of error.
//
//	flax.MustBindAll(flagSet, &flags1, &flags2)
//
// # Supported Types
//
// This package can bind a field of any of the default types supported by the
// standard [flag] package, including any type that implements the [flag.Value]
// interface.
//
// In addition, a field whose type implements the [encoding.TextMarshaler] and
// [encoding.TextUnmarshaler] interfaces can be bound as a flag, using the
// UnmarshalText method to "set" the flag and using its MarshalText method to
// render the value of the field.
//
// If a field implements both [flag.Value] and the text marshaling interfaces,
// the flag value implementation is used.
package flax

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MustBind binds the flaggable fields of v to fs, or panics. The concrete type
// of v must be a pointer to a value of struct type.  This function is intended
// for use in program initialization; callers who need to check errors should
// call [Check] and [Fields.Bind].
func MustBind(fs *flag.FlagSet, v any) {
	fi, err := Check(v)
	if err != nil {
		panic("check flags: " + err.Error())
	}
	fi.Bind(fs)
}

// MustBindAll is shorthand for calling MustBind(fs, v) for each v in vs.
func MustBindAll(fs *flag.FlagSet, vs ...any) {
	for _, v := range vs {
		MustBind(fs, v)
	}
}

// MustCheck constructs a [Fields] value from the flaggable fields of v, or
// panics.  This function is intended for use in program initialization;
// callers who need to check errors should call [Check] directly.
func MustCheck(v any) Fields {
	fields, err := Check(v)
	if err != nil {
		panic("check flags: " + err.Error())
	}
	return fields
}

// Check constructs information about the flaggable fields of v, whose concrete
// type must be a pointer to a value of struct type.
//
// Check reports an error if v has the wrong type, or if it does not define any
// flaggable fields.  An exported field of v is flaggable if it is of a
// compatible type and has a struct tag with the following form:
//
//	flag:"name[,default=V],Usage string"
//
// The name and usage string are required. Unexported fields and fields without
// a flag tag are ignored. If V contains commas, enclose it in 'single quotes',
// for example:
//
//	flag:"name,default='a, b',Usage string"
//
// To escape a quote, double it ("”").  If the default value begins with "$",
// it is interpreted as the name of an environment variable to read for the
// default. Double the "$" to escape this interpretation.
//
// If the default value is "*", it means to use the existing value of the
// target field as the default, rather than a zero. Use "**" to escape this
// meaning to get a literal star.
//
// As an alternative, a default may be specified separately via:
//
//	flag-default:"a, b"
//
// The two forms are mutually exclusive, even if the values are identical.
//
// Compatible types include bool, float64, int, int64, string, [time.Duration],
// uint, and uint64, as well as any type implementing the [flag.Value] interface
// or the [encoding.TextMarshaler] and [encoding.TextUnmarshaler] interfaces.
func Check(v any) (Fields, error) {
	if v == nil {
		return nil, errors.New("value is nil")
	}
	rp := reflect.ValueOf(v)
	if rp.Kind() != reflect.Ptr {
		return nil, errors.New("value is not a pointer")
	}
	rv := rp.Elem()
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("value is not a struct")
	}
	rt := rv.Type()

	var fields Fields
	for i := 0; i < rt.NumField(); i++ {
		fi, err := parseFieldValue(rt.Field(i), rv.Field(i))
		if err == errSkipField {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("field %q: %w", rt.Field(i).Name, err)
		}
		fields = append(fields, fi)
	}
	if len(fields) == 0 {
		return nil, errors.New("no flaggable fields")
	}
	return fields, nil
}

// Fields records information about the flaggable fields of a struct type.  Use
// the Bind method to attach flags to the corresponding fields.
type Fields []*Field

// Bind attaches the flags defined by f to the given flag set.
func (f Fields) Bind(fs *flag.FlagSet) {
	for _, fi := range f {
		fi.Bind(fs)
	}
}

// Flag returns the first entry in f whose flag name matches s, or nil if no
// such entry exists.
func (f Fields) Flag(s string) *Field {
	for _, fi := range f {
		if fi.Name == s {
			return fi
		}
	}
	return nil
}

// A Field records information about a single flaggable field in a struct type.
// The caller can modify the Name and Usage fields if desired before binding
// the flag to a [flag.FlagSet].
type Field struct {
	Name, Usage string // name and usage text (required)

	env    string // environment variable from which default is read
	dvalue any    // concrete type depends on target
	target any    // pointer to target field value
}

// Bind registers the field described by f in the given flag set.
func (fi *Field) Bind(fs *flag.FlagSet) {
	usage := fi.Usage
	if fi.env != "" {
		usage += fmt.Sprintf(" [env: %s]", fi.env)
	}
	switch t := fi.target.(type) {
	case flag.Value:
		fs.Var(t, fi.Name, usage)

	case textFlag:
		fs.TextVar(t, fi.Name, fi.dvalue.(textFlag), usage)

	case *bool:
		fs.BoolVar(t, fi.Name, fi.dvalue.(bool), usage)

	case *float64:
		fs.Float64Var(t, fi.Name, fi.dvalue.(float64), usage)

	case *int:
		fs.IntVar(t, fi.Name, fi.dvalue.(int), usage)

	case *int64:
		fs.Int64Var(t, fi.Name, fi.dvalue.(int64), usage)

	case *string:
		fs.StringVar(t, fi.Name, fi.dvalue.(string), usage)

	case *time.Duration:
		fs.DurationVar(t, fi.Name, fi.dvalue.(time.Duration), usage)

	case *uint:
		fs.UintVar(t, fi.Name, fi.dvalue.(uint), usage)

	case *uint64:
		fs.Uint64Var(t, fi.Name, fi.dvalue.(uint64), usage)

	default:
		panic(fmt.Sprintf("cannot flag type %T", t))
	}
}

// Env reports the name of the environment variable used as the default value
// for fi. It returns "" if the field does not use an environment variable.
func (fi *Field) Env() string { return fi.env }

var errSkipField = errors.New("skip this field")

func parseFieldValue(ft reflect.StructField, fv reflect.Value) (*Field, error) {
	if !ft.IsExported() {
		return nil, errSkipField // unexported fields are not considered
	}
	tag, ok := ft.Tag.Lookup("flag")
	if !ok {
		return nil, errSkipField // un-flagged fields are not considered
	}
	name, dstring, usage, err := parseFieldTag(tag)
	if err != nil {
		return nil, err
	}
	if dtag, ok := ft.Tag.Lookup("flag-default"); ok {
		if dstring != "" {
			return nil, fmt.Errorf("field %q default tag and string are both set", ft.Name)
		}
		dstring = dtag
	}

	vptr := fv.Addr().Interface()
	info := &Field{
		Name:   name,
		Usage:  usage,
		target: vptr,
	}

	// Check for compatible type.
	switch t := vptr.(type) {
	case *bool:
		d, err := parseDefault(info, dstring, *t, strconv.ParseBool)
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case *float64:
		d, err := parseDefault(info, dstring, *t, func(s string) (float64, error) {
			return strconv.ParseFloat(s, 64)
		})
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case *int:
		d, err := parseDefault(info, dstring, *t, strconv.Atoi)
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case *int64:
		d, err := parseDefault(info, dstring, *t, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		})
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case *string:
		// We call parseDefault here for the env handling; it can't fail.
		d, _ := parseDefault(info, dstring, *t, func(s string) (string, error) {
			return s, nil
		})
		info.dvalue = d

	case textFlag:
		_, err := parseDefault(info, dstring, nil, func(s string) (any, error) {
			return nil, t.UnmarshalText([]byte(s))
		})
		if err != nil {
			return nil, err
		}
		info.dvalue = t

	case *time.Duration:
		d, err := parseDefault(info, dstring, *t, time.ParseDuration)
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case *uint:
		d, err := parseDefault(info, dstring, *t, func(s string) (uint, error) {
			u, err := strconv.ParseUint(s, 10, 64)
			return uint(u), err
		})
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case *uint64:
		d, err := parseDefault(info, dstring, *t, func(s string) (uint64, error) {
			return strconv.ParseUint(s, 10, 64)
		})
		if err != nil {
			return nil, err
		}
		info.dvalue = d

	case flag.Value:
		_, err := parseDefault(info, dstring, nil, func(s string) (any, error) {
			return nil, t.Set(s)
		})
		if err != nil {
			return nil, err
		}
		info.dvalue = t

	default:
		return nil, fmt.Errorf("type %T is not flag compatible", t)
	}

	return info, nil
}

// Quoted default: ' ... ', allows "," and single quotes (as ”).
// Plain default:  ..., no "," or single quotes.
var defaultRE = regexp.MustCompile(`^('(?:[^']|'')*'|[^,']*),(.*)$`)

func parseFieldTag(s string) (name, dstring, usage string, _ error) {
	// Simple format: "name,usage"
	// Default format: "name,default=V,usage"

	name, usage, ok := strings.Cut(s, ",")
	if !ok {
		return "", "", "", fmt.Errorf("invalid flag tag format %q", s)
	}

	if d, ok := strings.CutPrefix(usage, "default="); ok {
		m := defaultRE.FindStringSubmatch(d)
		if m == nil {
			return "", "", "", fmt.Errorf("invalid default format %q", d)
		}
		dstring, usage = m[1], m[2]
		if strings.HasPrefix(dstring, "'") {
			dstring = strings.ReplaceAll(dstring[1:len(dstring)-1], "''", "'") // remove 'quotations'
		}
	}
	if name == "" {
		return "", "", "", errors.New("empty flag name")
	}
	return
}

func parseDefault[T any](f *Field, s string, self T, parse func(string) (T, error)) (T, error) {
	if strings.HasPrefix(s, "$$") {
		s = s[1:] // unescape leading "$"
	} else if env, ok := strings.CutPrefix(s, "$"); ok {
		f.env = env
		s = os.Getenv(env) // read default from environment
	} else if s == "**" {
		s = "*"
	} else if s == "*" {
		return self, nil
	}
	var zero T
	if s == "" {
		return zero, nil
	}
	v, err := parse(s)
	if err != nil {
		return zero, fmt.Errorf("invalid default for %q: %w", f.Name, err)
	}
	return v, nil
}

type textFlag interface {
	MarshalText() ([]byte, error)
	UnmarshalText([]byte) error
}
