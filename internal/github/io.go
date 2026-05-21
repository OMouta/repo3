package github

import "io"

func closeIgnore(closer io.Closer) {
	_ = closer.Close()
}
