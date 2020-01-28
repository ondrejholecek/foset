# Foset internal plugin: merge

This plugin allows to load some additional data for some or all the sessions. Such additional data must be saved in another
text file identified by `file` parameter. The file must be composed of columns divided by space (or some other separator
which can be specified using `sep` parameter). 

The session matching can be only done by session serial number. That means that one of the columns must contain the serial
number. By default such column should be named (see bellow for naming columns) `serial`, but different name can be specified
using the `key` parameter.

## Columns naming

Each column in the additional file must have the name, which is not specified inside the file but rather in the plugin
parameters string.

There are special parameters called by the column position in the file. For the first column the parameter is called `1`, 
for second `2`, etc. The parameter value is the name of the column. As described above, at least one of the names must be
`serial` (or another name selected by the `key` parameter).

The name can have an optional suffix specifying the type of the column. Following types are recognized:

| type | meaning                             |
|------|-------------------------------------|
| %s   | (default) simple string             |
| %d   | natural number (positive integer)   |
| %x   | same as %d but loads as hexadecimal |
| %f   | positive floating point number      |

Column names are then used as the names of the custom parameters. Identified by `custom` keyword in filter strings (`-f`, 
example `-f 'custom department = IT'`) or in the output formatter strings (`-o`, example `-o '${custom|department}'`).

## Example

To load data from a file `additional.txt` which contains the column delimited by space, where the first column
is the session serial number in hexadecimal format (can have optional `0x` prefix), the second column is a text
string called `department` and the third is positive integer called `floor`:

`-p 'merge|file=additional.txt,1=serial%x,2=department,3=floot%d'`

The file itself can look like this:

```
68fc3c2e IT 1
68c0004b IT 1
68ffb63a Marketing 2
68ff9372 Marketing 2
68ff9370 Sales 3
```

To merge it with the session list:

```
$ foset -r sessions.txt -p 'merge|file=additional.txt,1=serial%x,2=department,3=floor%d'
```

With the command above the session data and additinal data are merged, but it is not useful because we didn't use the 
custom variables anywhere.

Let's append the `departure` name and `floor` after the basic information about the session:

```
$ foset -r sessions.txt -p 'merge|file=additional.txt,1=serial%x,2=department,3=floor%d' \
  -o '${default_basic} ${custom|department} ${custom|floor}'

[...]
651584a9:   0/11    UDP         SEEN/SEEN        10.109.52.137:46160   -> 10.220.201.53:53      - 0
68fbb2e7:   0/45    UDP         SEEN/UNSEEN      10.109.16.20:23316    -> 62.209.40.72:8888     - 0
68fba46a:   0/45    UDP         SEEN/UNSEEN      10.109.16.20:23316    -> 62.209.40.73:8888     - 0
68fbb2e6:   0/45    UDP         SEEN/UNSEEN      10.109.16.20:23316    -> 62.209.40.74:8888     - 0
68c0004b:   0/36    TCP         NONE/ESTABLISHED 172.26.48.39:7071     -> 10.109.3.29:5008      IT 1
68fc3c2e:   0/7     TCP         NONE/ESTABLISHED 10.109.51.171:20549   -> 10.5.53.146:514       IT 1
68ffb63a:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:33455     -> 8.254.13.1:53         Marketing 2
68ff9372:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:23432  -> 96.45.33.64:53        Marketing 2
68ff9370:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:23432  -> 96.45.33.65:53        Sales 3
68ffb406:   0/1     UDP         SEEN/UNSEEN      10.109.3.26:61152     -> 208.91.112.53:53      - 0
68ffa51d:   0/1     UDP         SEEN/UNSEEN      10.109.3.36:60682     -> 8.8.8.8:53            - 0
68ff946d:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:60912     -> 205.251.194.202:53    - 0
68ffa67a:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:54371     -> 2.22.230.129:53       - 0
[...]
```

We can see that all the sessions are still display, but only some of them matched the additional.txt data. The sessions
that didn't match contain empty values in the custom variables (which is `-` for strings and `0` for integers).

If we want to display only the sessions that are both in the sessions.txt file and in the addtional.txt file,
we need to specify also some filter (note that `serial` column is also include in the custom variable `serial` which
is completely independent on the regular session variable `serial`):

```
$ foset -r sessions.txt -p 'merge|file=additional.txt,1=serial%x,2=department,3=floor%d' \
  -o '${default_basic} ${custom|department} ${custom|floor}' -f 'custom serial'
  
68c0004b:   0/36    TCP         NONE/ESTABLISHED 172.26.48.39:7071     -> 10.109.3.29:5008      IT 1
68fc3c2e:   0/7     TCP         NONE/ESTABLISHED 10.109.51.171:20549   -> 10.5.53.146:514       IT 1
68ffb63a:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:33455     -> 8.254.13.1:53         Marketing 2
68ff9372:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:23432  -> 96.45.33.64:53        Marketing 2
68ff9370:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:23432  -> 96.45.33.65:53        Sales 3
```

It is of course possible to filter even further based on the custom variables:

```
$ foset -r sessions.txt -p 'merge|file=additional.txt,1=serial%x,2=department,3=floor%d' \
  -o '${default_basic} ${custom|department} ${custom|floor}' -f 'custom serial and custom floor >= 2'
  
68ffb63a:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:33455     -> 8.254.13.1:53         Marketing 2
68ff9372:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:23432  -> 96.45.33.64:53        Marketing 2
68ff9370:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:23432  -> 96.45.33.65:53        Sales 3
```
  
