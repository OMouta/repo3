package s3api

import "io"

func closeIgnore(closer io.Closer) {
	_ = closer.Close()
}
