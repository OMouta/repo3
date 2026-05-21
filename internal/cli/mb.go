package cli

import "fmt"

func (c command) runMB(args []string) error {
	fs, opts, err := c.parseClientFlags("mb", args)
	if err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: repo3 mb <bucket>")
	}

	bucket := fs.Arg(0)
	if err := newClient(opts).CreateBucket(background(), bucket); err != nil {
		return err
	}
	if !opts.json {
		fmt.Fprintf(c.stdout, "created bucket %s\n", bucket)
	}
	return nil
}
