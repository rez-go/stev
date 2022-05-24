# stev

Structured configuration through environment variables.

There are similar projects out there, but we push further with this one
to solve some issues. The issues are:

- it's difficult to figure out what fields are available, what's
  the type or kind of expected value, the requirements etc.
- sometimes, there is out-of-sync between implementation
  and existing config template or existing config files.

This project solves both issues by providing an ability to generate
a config template, including the documentation for each field,
by querying the application executable so that the template
generated matches the implementation perfectly.

To try it out, we have an example application at `examples/main.go`.
It has dummy configuration structs. To generate its config template,
run:

```sh
$ go run examples/main.go env_file_template
```

The template will be printed out to the stdout.
