package cli

import (
	"fmt"

	"repo3/internal/client"
)

func (c command) runRM(args []string) error {
	fs, opts, err := c.parseClientFlags("rm", args)
	if err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: repo3 rm s3://bucket/key")
	}

	s3Path, err := client.ParseS3Path(fs.Arg(0))
	if err != nil {
		return err
	}
	if s3Path.Key == "" {
		return fmt.Errorf("target must include an object key")
	}
	if err := newClient(opts).DeleteObject(background(), s3Path.Bucket, s3Path.Key); err != nil {
		return err
	}
	if !opts.json {
		_, err := fmt.Fprintf(c.stdout, "deleted %s\n", fs.Arg(0))
		return err
	}
	return nil
}
