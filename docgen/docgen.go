package docgen

import (
	"fmt"
	"io"

	"github.com/rez-go/stev"
)

type EnvTemplateOptions struct {
	RegionTitle string
	FieldPrefix string
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

	if opts.RegionTitle != "" {
		fmt.Fprintf(writer, "#region %s\n", opts.RegionTitle)
	}

	for _, fd := range fieldDocs {
		fmt.Fprintf(writer, "\n")
		if fd.Description != "" {
			fmt.Fprintf(writer, "# %s\n#\n", fd.Description)
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

	if opts.RegionTitle != "" {
		fmt.Fprintf(writer, "\n#endregion\n")
	}

	return nil
}
