package gcfg

import (
	"fmt"

	"github.com/launchdarkly/gcfg/token"
)

// ErrorAction is a type that is returned by an ErrorHandler function to indicate what to
// do about a problem.
type ErrorAction int

const (
	// ErrorActionNone means that the ErrorHandler has no opinion about the error. It may
	// be handled by a subsequent error handler, but if not, it will be ignored.
	ErrorActionNone ErrorAction = iota

	// ErrorActionStop means that the ErrorHandler considers the error to be fatal, so gcfg
	// should stop processing and return the error to the caller.
	ErrorActionStop ErrorAction = iota

	// ErrorActionSuppress means that the ErrorHandler wants gcfg to ignore the error. It
	// will not be passed to any subsequent error handlers.
	ErrorActionSuppress ErrorAction = iota
)

// ErrorLocation describes a location where an error occurred. This is provided as part
// of a TargetNotFoundError, ValueError, or InvalidContainerError.
//
// If Field is non-empty, then the problem is with the specified field; Section and
// Subsection will also be set.
//
// If Field is empty, then the problem is with the specified Section and/or Subsection.
//
// If all three fields are empty (for InvalidContainerError), the problem is with the
// target data structure that was passed to the Read function.
type ErrorLocation struct {
	Section    string
	Subsection string
	Field      string
}

func (l ErrorLocation) describeLocation() string {
	switch {
	case l.Field != "":
		return fmt.Sprintf("section %q subsection %q variable %q", l.Section, l.Subsection, l.Field)
	case l.Subsection != "":
		return fmt.Sprintf("section %q subsection %q", l.Section, l.Subsection)
	default:
		return fmt.Sprintf("section %q", l.Section)
	}
}

// ParseError is an error that indicates that the configuration data had invalid syntax.
// The Err value describes what the specific problem was.
//
// This type of error always causes gcfg to stop and return the error to the caller, since
// parsing cannot continue if the input is malformed.
//
// Some kinds of syntax errors cause gcfg to return a scanner.Error instead of a ParseError.
type ParseError struct {
	token.Position
	Err error
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.Position, e.Err)
}

// TargetNotFoundError is an error that indicates a section name or field name in the
// configuration did not exist in the target data structure.
//
// By default, gcfg does not report this type of error. You can use ErrorHandler, with either
// StopOnTargetNotFound or a custom handler, to change that behavior.
type TargetNotFoundError struct {
	ErrorLocation
}

func (e TargetNotFoundError) Error() string {
	switch {
	case e.Field != "":
		return "invalid variable: " + e.describeLocation()
	case e.Subsection != "":
		return "invalid subsection: " + e.describeLocation()
	default:
		return "invalid section: " + e.describeLocation()
	}
}

// ValueError is an error that indicates that a value in the configuration file was not in a
// valid format for the corresponding part of the target data structure. For instance, this
// could mean that an int field was set to a non-numeric string. The Err value indicates
// what the specific problem was, and the ErrorLocation fields describe where it occurred.
//
// By default, if gcfg encounters this type of error, it stops parsing and returns the error.
// If you use the ErrorHandler option, it will pass this error to the handler function which
// can determine what to do.
type ValueError struct {
	ErrorLocation
	Err error
}

func (e ValueError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.describeLocation())
}

// InvalidContainerError is an error that indicates that a section within the target data
// structure was not a valid container type that gcfg can set values in. For instance, this
// could mean it was not a struct or map, or that it was a map with unsupported key or value
// types. The Message field describes the problem more specifically.
//
// By default, if gcfg encounters this type of error, it panics. If you use the ErrorHandler
// option, it will pass this error to the handler function which can determine what to do.
type InvalidContainerError struct {
	ErrorLocation
	Message string
}

func (e InvalidContainerError) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.describeLocation())
}

// StopOnTargetNotFound can be used with ErrorHandler to change gcfg's behavior regarding
// unrecognized names.
//
// By default, if a section or field in the configuration does not correspond to anything
// in the target data structure, gcfg skips it. If you specify ErrorHandler(StopOnTargetNotFound),
// it will instead report this as a TargetNotFoundError.
//
// If you want to customize error-reporting behavior in other ways, use ErrorHandler with a
// custom function.
//
//     err := gcfg.ReadFileInto(&configStruct, fileName,
//         gcfg.ErrorHandler(gcfg.StopOnTargetNotFound))
func StopOnTargetNotFound(e error) ErrorAction {
	if _, ok := e.(TargetNotFoundError); ok {
		return ErrorActionStop
	}
	return ErrorActionNone
}

// defaultErrorHandler is the fallback handler that the Read functions use if there is no
// custom handler, or if the custom handler(s) all returned ErrorActionNone.
func defaultErrorHandler(e error) ErrorAction {
	switch e := e.(type) {
	case TargetNotFoundError:
		// Default for this type of error is to skip it.
		// A LaunchDarkly addition to the standard gcfg behavior is that these errors are also logged to
		// the console.
		fmt.Printf("gcfg: %s\n", e)
		return ErrorActionSuppress
	case ValueError:
		// Default for this type of error is to stop; the error will be returned to the caller.
		return ErrorActionStop
	case InvalidContainerError:
		// Default behavior for this type of error is to panic.
		panic(e)
	default:
		// Error handlers will normally receive one of the above types of errors; stop for any other type.
		return ErrorActionStop
	}
}
