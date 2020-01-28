# FortiSession

Go packages (libraries) for parsing FortiGate session list output.

Please be aware that this software is licensed by [CC BY-ND 4.0 license](https://creativecommons.org/licenses/by-nd/4.0/) - make sure you understand it before you decide to use this code.

For detailed documentation use GoDoc or check the source code of the [Foset program using this library](https://github.com/ondrejholecek/foset).

There are following (sub-) packages in this repository:

## fortisession

Contains structures to hold session data and parser functions to retrieve them. 

The main function here is `Parse` which expects byte slice input `data` containg one full session starting with `session info: proto=...`. The `requested` is pointer to the structure specifying which fields (group of fields actually) are to be parsed - this is to improve parsing speed when only some fields are required (which is very common).

```
func Parse(data []byte, requested *SessionDataRequest) *Session
```

It returns point to `Session` structure containg all the fields in groups (or `nil` if the group wasn't parsed).


## fortisession/forticonditioner

FortiConditioner is the package that analyses the text filter and then decides whether the session matches it or not. Filter string is described in [Conditions format file](forticonditioner/condition_format.md).

It must be initialized with `Init` function which accepts the filter string in `filter` parameter and pointer to the structure saying which fields need to be parsed - the `Init` will enable those field groups that are needed (the filter uses them).

```
func Init(filter string, request *fortisession.SessionDataRequest) *Condition
```

It returns pointer to `Condition` structure, which then needs to be used with `Matches()` function to decide whether the filter matches the session give in `session` parameter or not.

```
func (cond *Condition) Matches(session *fortisession.Session) bool
```

## fortisession/fortiformatter

FortiFormatter is the package to return the session fields in the selected format. Format string is described in [Output format file](fortiformatter/output_format.md).

Again, it must be initialized with `Init` function which accepts the output format string in `format` parameter and pointer to `request` structure, where the requited field groups will be enabled based on the required outputs.

```
func Init(format string, request *fortisession.SessionDataRequest) (*Formatter, error)
```

It returns pointer to `Formatter` structure, which then needs to be used with `Format()` function to return the string based on the format string with the session fields.

```
func (f *Formatter) Format(session *fortisession.Session) string
```
