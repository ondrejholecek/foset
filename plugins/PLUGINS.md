# Plugins

Foset core offers only functionality for parsing the session file, filtering it and formatting it on the standard output.
More advanced features are implemented using plugin system. 

There are two types of plugins, loaded either with `-p` or with `-P` command line option. In both cases the parameter
has following format: `-p name|parameter1=value1,parameter2=value2,noValueParameter`. The `name` identifies the plugin
and its specific parameters follow after `|`. Each parameter can have a value (like `parameter1` has `value1` and `parameter2`
has `value2`), but it also can only be a boolean parameter - just enabling some feature without the need for value
(like `noValueParameter` in the example above).

## External plugins

External plugins are loaded using '-P' (capital P). In that case the `name` can either be a full path to the plugin
(which is .so file) or just a plugin name without any path and file extension. 

In the later case the environment variable `FOSET_PLUGINS` is used to locate the file which must have the same name
and .so extension and be in one of the directories specifed by the variable (`PATH`-like behavior is expected, ie. 
there can be more diretories separated by `:`).

If the `FOSET_PLUGINS` environment variable does not exist or it is empty, then the directory `.foset/plugins` in the
directory found in `HOME` environment variable is used.

Source code for external plugins may not be published anywhere and such plugins can only be delivered in their
binary form. The compatibility with the main Foset binary is not guaranteed between versions, however it should
be compatible as long as the internal Session structure is not changed. In case of incompatibility, the system
does not allow to use the incompatible external plugin.


## Internal plugins 

Internal plugins are implemented inside the Foset source code and are automatically included in all the binary builds. 
Such plugins are loaded using `-p` option and the name is always the plain plugin name without any path or file
extension.

Currently recognized (and described plugins):
- [merge](/plugins/merge/merge.md): allows to merge a text file with additional data about the sessions that is loaded
into specified custom variables
- [indexmap](/plugins/indexmap/indexmap.md): with provided outputs of some addtional FortiGate commands it translates
the VDOM and interface indexes into their real names
- [stats](/plugins/stats/stats.md): creates a local webpage with a lot of top-X statistics about the session table in
the form of beautiful graphs
