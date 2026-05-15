package parser

import "errors"

var (
	ErrArchiveNotFound       = errors.New("archive not found")
	ErrInvalidArchive        = errors.New("invalid or corrupted archive")
	ErrRequiredFilesNotFound = errors.New("required files not found in archive (.db_csv and .sharp_an_info)")
	ErrInvalidCSVFormat      = errors.New("invalid csv format")
	ErrInvalidNodeRecord     = errors.New("invalid node record")
	ErrPathTraversal         = errors.New("path traversal detected")
)
