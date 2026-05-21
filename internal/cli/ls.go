package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"repo3/internal/client"
)

func (c command) runLS(args []string) error {
	fs, opts, err := c.parseClientFlags("ls", args)
	if err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: repo3 ls s3://bucket[/prefix]")
	}

	s3Path, err := client.ParseS3Path(fs.Arg(0))
	if err != nil {
		return err
	}
	objects, err := newClient(opts).ListObjects(background(), s3Path.Bucket, s3Path.Key)
	if err != nil {
		return err
	}

	if opts.json {
		enc := json.NewEncoder(c.stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(objects)
	}

	tw := tabwriter.NewWriter(c.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SIZE\tLAST MODIFIED\tKEY")
	for _, obj := range objects {
		fmt.Fprintf(tw, "%d\t%s\t%s\n", obj.Size, obj.LastModified.Format("2006-01-02 15:04"), obj.Key)
	}
	return tw.Flush()
}
