# File descriptor provider

The is not used when running Foset manually, but rather when it is started from another program (like `foset-webui`) and the "file" data need to be provided without creating any temporary files.

## Basic info

- schema: `fd://`
- rest: number specifying the file descriptor

## Examples

- `fd://0` - standard input on Linux
- `fd://1` - standard output on Linux
- `fd://4` - custom pipe opened by the calling program, the direction depends on how it was opened

## Output provider

It can be also used as output provider, but the pipes are usually unidirectional so it is necessary to open it correctly. 
