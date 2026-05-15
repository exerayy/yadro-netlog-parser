package parser

import "context"

type DB interface {
	SaveLogData(ctx context.Context, data *LogData) (int64, error)
}

type IParser interface {
	Parse(ctx context.Context, path string) (int64, error)
}
