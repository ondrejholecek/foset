# File provider

## Basic info

- schema: `file://`
- rest: path to the file

## Examples

- `file:///tmp/sessions.gz` - note `///` - first two are part of the schema, and the third one is path of the file path
- `file://Jan20/sessions.gz - relative path from the current directory (note only two `//`)
- `file://sessions.gz` - file in current directory

## Default provider

Because `file` is default provider, any file name without the schema is considered as if it had `file://` schema at the beginning:

- `session.gz`
- `~/tmp/cores.gz`

## Output provider

The `file` provider can also be used as the output provider where it makes sense.
