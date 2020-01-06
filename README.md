# FOrtigate SEssion Tool

This tool utilizes the [FortiSession library](https://github.com/ondrejholecek/fortisession) to parse the text output of `diagnose sys session list` command.

It can read either plain-text files (Putty log output or Linux "script" command output) specified with `-f` or `--file` command line parameter, or the same files compressed with Gzip (add also `-g`).

The session fields can be displayed in different formats controlled by "format string" given as command line parameter (`-o` or `--output`). To learn how to specify the output, see [Output format description](https://github.com/ondrejholecek/fortisession/blob/master/fortiformatter/output_format.md).

All sessions can be displayed or only interesting sessions can be selected using the "filter string" given as another command line parameters (`-f` or `--filter`). To learn how to use the conditions to select the right sessions, see [Condition format description](https://github.com/ondrejholecek/fortisession/blob/master/forticonditioner/condition_format.md).

