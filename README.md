# stev

Structured configuration through environment variables.

There are similar projects out there, but we push it further with this one
to solve some issues. The issues are:

- it's difficult to figure out what fields are available, what's
  the type or kind of expected value, the requirement constraint etc.
- the existing template or config files could be out-of-sync with what was
  implemented in the executable which could cause confusion as why a field
  is not working or behaves differently.

This package solves both issues by providing an ability to generate
a config template, including the detailed documentation, by querying
the application executable so that the template generated matches
the implementation perfectly for the said executable.

To try it out, we have an example application at `examples/basic_docgen.go`.
It has dummy configuration structs. To generate its config template,
run:

```sh
$ go run examples/basic_docgen.go env_file_template
```

The template will be printed out to the stdout.

Here, the argument `env_file_template` is the command for the application
to generate its accurate configuration template. Your application could use
other command or method to trigger it. The key is to call
`docgen.WriteEnvTemplate`.

For more complex example, look at [kadisoka-framework](https://github.com/kadisoka/kadisoka-framework/blob/master/apps/iam-standalone-server/etc/iam-server/secrets/config.env.example).

Summary of features

- Loading environment variable values into a struct
- Nested configuration support which allows configurations for deeper
  functionalities to be exposed with proper namespacing
- Support for generating config template based on the
  enabled modules built into the executable
- Support for generating documentation for available enumerated values in
  config template
