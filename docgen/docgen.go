package docgen

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"github.com/rez-go/stev"
)

type EnvTemplateOptions struct {
	FieldPrefix string

	// By default, the fields are sorted alphabetically by the keys. If it's
	// prefered to keep their order as found in the structs, set this option
	// to true.
	OriginalOrdering bool

	// By default, there won't be values in the generated template. If this
	// option is set to true, then any field in the skeleton which value is
	// set, that value will be used as the value of the field in the generated
	// template.
	IncludeSkeletonValues bool
}

func WriteEnvTemplate(
	writer io.Writer,
	skeleton interface{},
	opts EnvTemplateOptions,
) error {
	fieldDocs, err := stev.Docs(opts.FieldPrefix, skeleton)
	if err != nil {
		panic(err)
	}

	if !opts.OriginalOrdering {
		sort.Slice(fieldDocs, func(i, j int) bool {
			return strings.Compare(fieldDocs[i].LookupKey, fieldDocs[j].LookupKey) < 0
		})
	}

	for _, fd := range fieldDocs {
		fmt.Fprintf(writer, "\n")
		if fd.Description != "" {
			descLines := strings.Split(wordwrap.WrapString(fd.Description, 72), "\n")
			for _, l := range descLines {
				fmt.Fprintln(writer, "#", l)
			}
			fmt.Fprintln(writer, "#")
		}
		if fd.Required {
			fmt.Fprintf(writer, "# required\n")
		}
		fmt.Fprintf(writer, "# type: %s\n", fd.DataType)
		if !opts.IncludeSkeletonValues && fd.Value != "" {
			fmt.Fprintf(writer, "#  def: %s\n", fd.Value)
		}
		fmt.Fprintf(writer, "# path: %s\n", fd.Path)
		if opts.IncludeSkeletonValues {
			fmt.Fprintf(writer, "%s=%s\n", fd.LookupKey, fd.Value) //TODO: escape? quote?
		} else {
			fmt.Fprintf(writer, "%s=\n", fd.LookupKey)
		}
	}

	return nil
}
