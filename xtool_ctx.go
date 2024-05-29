package main

import "context"

type contextKey string

var (
	contextKeyErrPrintln = contextKey("errPrintln")
	contextKeyErrPrintf  = contextKey("errPrintf")
	contextKeyAppConfig  = contextKey("appConfig")
)

func CtxWthErrPrintln(ctx context.Context, errPrintln func(...interface{})) context.Context {
	return context.WithValue(ctx, contextKeyErrPrintln, errPrintln)
}

func CtxWthErrPrintf(ctx context.Context, errPrintf func(string, ...interface{})) context.Context {
	return context.WithValue(ctx, contextKeyErrPrintf, errPrintf)
}

func CtxWthAppConfig(ctx context.Context, appConfig AppConfig) context.Context {
	return context.WithValue(ctx, contextKeyAppConfig, appConfig)
}

func ErrPrintln(ctx context.Context, args ...interface{}) {
	if errPrintln, ok := ctx.Value(contextKeyErrPrintln).(func(...interface{})); ok {
		errPrintln(args...)
	} else {
		panic("ErrPrintln: no errPrintln in context")
	}
}

func ErrPrintf(ctx context.Context, format string, args ...interface{}) {
	if errPrintf, ok := ctx.Value(contextKeyErrPrintf).(func(string, ...interface{})); ok {
		errPrintf(format, args...)
	} else {
		panic("ErrPrintf: no errPrintf in context")
	}
}

func ErrPrint(ctx context.Context, err error) {
	ErrPrintln(ctx, err.Error())
}

func AppConfigFromCtx(ctx context.Context) AppConfig {
	if appConfig, ok := ctx.Value(contextKeyAppConfig).(AppConfig); ok {
		return appConfig
	} else {
		panic("AppConfigFromCtx: no appConfig in context")
	}
}
