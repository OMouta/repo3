package client

import "io"

func closeIgnore(closer io.Closer) {
	_ = closer.Close()
}
