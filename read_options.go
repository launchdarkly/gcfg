package gcfg

type readOptions struct {
	errorHandlers        []func(error) ErrorAction
	stopOnTargetNotFound bool
}

// ReadOption is a common interface for optional parameters that can be passed to
// the Read functions.
type ReadOption interface {
	apply(*readOptions)
}

// ErrorHandler is an option for the Read functions which causes the specified function(s)
// to be called whenever the configuration contains a name or value that is not valid for
// the target data structure.
//
// There are several types of errors: see TargetNotFoundError, ValueError, and
// InvalidContainerError. For any of these, gcfg will call the specified handler function(s)
// in order. If a handler returns ErrorActionStop, gcfg will stop and return the error to
// the caller. If a handler returns ErrorActionSuppress, gcfg will ignore the error.
// Otherwise it will proceed to the next handler, if any, and if there are no more it will
// fall back to the default error-handling behavior.
//
// The default error-handling behavior is that TargetNotFoundError is printed to the console
// and then ignored; ValueError causes gcfg to stop and return the error, and
// InvalidContainerError causes a panic.
//
// Two kinds of errors cannot be handled with ErrorHandler:
//
// 1. Errors that are due to an incorrectly formatted file, so that parsing cannot continue,
// always cause the Read functions to immediately return the error (as a ParseError).
//
// 2. Passing a target interface{} value that is not a struct pointer to the Read functions
// always causes a panic.
//
//     func logAndSkipValueErrors(e error) gcfg.ErrorAction {
//         if _, ok := e.(gcfg.ValueError); ok {
//             fmt.Printf("warning: %v\n", e)
//             return gcfg.ErrorActionSuppress
//         }
//         return gcfg.ErrorActionNone
//     }
//
//     err := gcfg.ReadFileInto(&configStruct, fileName,
//         gcfg.ErrorHandler(logAndSkipValueErrors))
func ErrorHandler(handler ...func(error) ErrorAction) ReadOption {
	return readOptionErrorHandlers(handler)
}

type readOptionErrorHandlers []func(error) ErrorAction

func (o readOptionErrorHandlers) apply(ro *readOptions) {
	ro.errorHandlers = o
}
