# Input providers

Any input data that comes in a form of a file, can actually come from any other supported source. 

Some input providers do not need any special configuration and it is enough to use the right schema (see bellow) but others need also some pre-configuration (like SSH host, username, password). Such configuration is passed in command line after `-i` parameter and looks like `name|parameter1=value1,parameter2=value2,noValueParameter`, where name is the name of the input provider this configuration is for and parameters can be either with values (`parameter1` with `value1`) or just as enabling switches (`noValueParameter`).

## Selecting the provider

The provider for each file is selected using a schema which is at the beginning of the file name and it is similar to URL. After the schema, there is data specific to each provider. Schema can be one of following:

- `ssh://` - [SSH input provider](/iproviders/ssh/) - connects to FortiGate using SSH and executes a command whose output is used as "inputfile"
- `fd://` - [File descriptor provider](/iproviders/fd/) - "inputfile" data is read from specified file descriptor which has to be passed to Foset by the calling program
- `file://` - [File provider](/iproviders/file/) - regular file provider, the string that follows is path to the file

## Default provider

If no schema is specified, the `file` schema is assumed and the "filename" is really a file name (or path to the file).
