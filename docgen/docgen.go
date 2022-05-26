package docgen

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"github.com/rez-go/stev"
)

type EnvTemplateWriteOptions struct {
	FieldPrefix string

	// By default, the fields are sorted alphabetically by the keys. If it's
	// prefered to keep their order as found in the structs, set this option
	// to true.
	OriginalOrdering bool

	// If set to true, the path to each field will be printed in the output.
	ShowPaths bool
}

// WriteEnvTemplate writes the template passed as `skeleton` through `writer`.
func WriteEnvTemplate(
	writer io.Writer,
	skeleton interface{},
	opts EnvTemplateWriteOptions,
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
		if len(fd.AvailableValues) > 0 {
			fmt.Fprintf(writer, "# available values:\n")

			enumVals := make([]string, len(fd.AvailableValues))
			i := 0
			for k := range fd.AvailableValues {
				enumVals[i] = k
				i++
			}
			sort.Strings(enumVals)

			for _, enumVal := range enumVals {
				docs := fd.AvailableValues[enumVal]
				if docs.ShortDesc != "" {
					fmt.Fprintf(writer,
						"#   %s - %s\n",
						enumVal, docs.ShortDesc)
				} else {
					fmt.Fprintf(writer, "#   %s\n", enumVal)
				}
			}
		}
		if opts.ShowPaths {
			fmt.Fprintf(writer, "# path: %s\n", fd.Path)
		}
		if fd.Value != "" {
			fmt.Fprintf(writer, "# %s=%s\n", fd.LookupKey, fd.Value) //TODO: escape? quote?
		} else if fd.Required {
			fmt.Fprintf(writer, "%s=\n", fd.LookupKey)
		} else {
			fmt.Fprintf(writer, "# %s=\n", fd.LookupKey)
		}
	}

	return nil
}
