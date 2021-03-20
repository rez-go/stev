package docgen

import (
	"fmt"
	"io"

	"github.com/rez-go/stev"
)

func WriteEnvTemplate(writer io.Writer, envVarsPrefix string, skeleton interface{}) error {
	fieldDocs, err := stev.Docs(envVarsPrefix, skeleton)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(writer, "#region Config\n")

	for _, fd := range fieldDocs {
		fmt.Fprintf(writer, "\n")
		if fd.Description != "" {
			fmt.Fprintf(writer, "# %s\n#\n", fd.Description)
		}
		if fd.Required {
			fmt.Fprintf(writer, "# required\n")
		}
		fmt.Fprintf(writer, "# type: %s\n", fd.DataType)
		fmt.Fprintf(writer, "# path: %s\n", fd.Path)
		fmt.Fprintf(writer, "%s=%s\n", fd.LookupKey, fd.Value) //TODO: escape?
	}

	fmt.Fprintf(writer, "\n#endregion\n")
	return nil
}
