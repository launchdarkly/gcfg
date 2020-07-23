package gcfg

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/launchdarkly/gcfg/scanner"
	"github.com/launchdarkly/gcfg/token"
)

var unescape = map[rune]rune{'\\': '\\', '"': '"', 'n': '\n', 't': '\t'}

// no error: invalid literals should be caught by scanner
func unquote(s string) string {
	u, q, esc := make([]rune, 0, len(s)), false, false
	for _, c := range s {
		if esc {
			uc, ok := unescape[c]
			switch {
			case ok:
				u = append(u, uc)
				fallthrough
			case !q && c == '\n':
				esc = false
				continue
			}
			panic("invalid escape sequence")
		}
		switch c {
		case '"':
			q = !q
		case '\\':
			esc = true
		default:
			u = append(u, c)
		}
	}
	if q {
		panic("missing end quote")
	}
	if esc {
		panic("invalid escape sequence")
	}
	return string(u)
}

func readInto(config interface{}, fset *token.FileSet, file *token.File, src []byte, options ...ReadOption) error {
	var readOptions readOptions
	for _, o := range options {
		o.apply(&readOptions)
	}
	errorHandlers := append(readOptions.errorHandlers, defaultErrorHandler)

	processErrorAndMaybeStop := func(e error) bool {
		shouldStop := false
	CheckHandlers:
		for _, h := range errorHandlers {
			switch h(e) {
			case ErrorActionStop:
				shouldStop = true
				break CheckHandlers
			case ErrorActionSuppress:
				break CheckHandlers
			default:
				continue
			}
		}
		return shouldStop
	}

	var s scanner.Scanner
	var errs scanner.ErrorList
	s.Init(file, src, func(p token.Position, m string) { errs.Add(p, m) }, 0)
	sect, sectsub := "", ""
	pos, tok, lit := s.Scan()
	errfn := func(msg string) error {
		return ParseError{Position: fset.Position(pos), Err: errors.New(msg)}
	}
	for {
		if errs.Len() > 0 {
			return errs.Err()
		}
		switch tok {
		case token.EOF:
			return nil
		case token.EOL, token.COMMENT:
			pos, tok, lit = s.Scan()
		case token.LBRACK:
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			if tok != token.IDENT {
				return errfn("expected section name")
			}
			sect, sectsub = lit, ""
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			if tok == token.STRING {
				sectsub = unquote(lit)
				if sectsub == "" {
					return errfn("empty subsection name")
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					return errs.Err()
				}
			}
			if tok != token.RBRACK {
				if sectsub == "" {
					return errfn("expected subsection name or right bracket")
				}
				return errfn("expected right bracket")
			}
			pos, tok, lit = s.Scan()
			if tok != token.EOL && tok != token.EOF && tok != token.COMMENT {
				return errfn("expected EOL, EOF, or comment")
			}
			// If a section/subsection header was found, ensure a
			// container object is created, even if there are no
			// variables further down.
			err := set(config, sect, sectsub, "", true, "")
			if err != nil && processErrorAndMaybeStop(err) {
				return err
			}
		case token.IDENT:
			if sect == "" {
				return errfn("expected section header")
			}
			n := lit
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			blank, v := tok == token.EOF || tok == token.EOL || tok == token.COMMENT, ""
			if !blank {
				if tok != token.ASSIGN {
					return errfn("expected '='")
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					return errs.Err()
				}
				if tok != token.STRING {
					return errfn("expected value")
				}
				v = unquote(lit)
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					return errs.Err()
				}
				if tok != token.EOL && tok != token.EOF && tok != token.COMMENT {
					return errfn("expected EOL, EOF, or comment")
				}
			}
			err := set(config, sect, sectsub, n, blank, v)
			if err != nil && processErrorAndMaybeStop(err) {
				return err
			}
		default:
			if sect == "" {
				return errfn("expected section header")
			}
			return errfn("expected section header or variable declaration")
		}
	}
	panic("never reached")
}

// ReadInto reads gcfg formatted data from reader and sets the values into the
// corresponding fields in config.
//
// You may specify ReadOptions such as ErrorHandler if you want to modify the default
// reading behavior. See ErrorHandler for a description of error-handling behavior.
func ReadInto(config interface{}, reader io.Reader, options ...ReadOption) error {
	src, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	return readInto(config, fset, file, src, options...)
}

// ReadStringInto reads gcfg formatted data from str and sets the values into
// the corresponding fields in config.
//
// You may specify ReadOptions such as ErrorHandler if you want to modify the default
// reading behavior. See ErrorHandler for a description of error-handling behavior.
func ReadStringInto(config interface{}, str string, options ...ReadOption) error {
	r := strings.NewReader(str)
	return ReadInto(config, r, options...)
}

// ReadFileInto reads gcfg formatted data from the file filename and sets the
// values into the corresponding fields in config.
//
// You may specify ReadOptions such as ErrorHandler if you want to modify the default
// reading behavior. See ErrorHandler for a description of error-handling behavior.
func ReadFileInto(config interface{}, filename string, options ...ReadOption) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	src, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file := fset.AddFile(filename, fset.Base(), len(src))
	return readInto(config, fset, file, src, options...)
}
