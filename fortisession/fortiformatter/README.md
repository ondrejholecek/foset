# OUTPUT FORMAT

Variables are enclosed in `${...}` text(s) and those will be replaced by the
actual value. Any other text is kept intact.

Some variables also accept additional parameters after `:` and/or `|` characters.
Either one of them or both at the same time can be used.

Characters after `:` are treated as [format parameters in `fmt` module
](https://golang.org/pkg/fmt/#hdr-Printing) without the initial percentage sign.
If not specified the default value is `s` with couple of exceptions with `d` and `x`.
Keep in mind that some variables appearing as numbers are actually strings,
because sometimes it is necessary to show an invalid value (like "-").

Also some variables can be used both as string and number, giving different results
(like the IP protocol can be 6 if used as number but "TCP" if used as string).

Characters after `|` are modifiers specific to each variable. Those can do anything
with the original variable (like specifying the separator string for multi-value content).

Here are some of the possible options. See the table bellow for complete list.
```
${protocol}          // "TCP" (because protocol is string by default)
${protocol:-4s}      // "TCP "
${protocol:d}        // "6"
${protocol:4d}       // "   6"
${protocol:-4d}      // "6   "
${protocol:04d}      // "0006"
${dp:05d}            // "00053" (destination port)
```

One of the specific option the "vdom" has is mapping the VDOM number to its name:
```
${vdom}              // "2"
${vdom|2=test}'      // "2" (because vdom is number by default)
${vdom:10s|2=test}   // "      test"
${vdom:-10s|2=test}  // "test      "
```

We can obviously write some surrounding text:
```
There is a session from ${sap} to ${dap} in VDOM ${vdom} (${vdom:s|0=zero,1=one,2=two,3=three}).
```

Results in:

```
There is a session from 50.100.10.12:1400 to 8.8.8.8:8 in VDOM 1 (one).
There is a session from 100.50.20.33:34878 to 8.8.8.8:53 in VDOM 2 (two).
```

## All supported variables

And their options. * denotes the default type for the variable. If it is not specified on any line
for that variable name, it is the only supported format. If one numeric type is supported,
all other numeric types are supported as well.

|     name      | type | example output    | comments and possible modifiers                       | aliases               |
| ------------- | ---- | ----------------- | ----------------------------------------------------- | --------------------- |
| serial        | x *  | 76e6b788          | session serial number                                 |                       |
| serial        | d    | 1994831752        |                                                       |                       |
| proto         | s *  | TCP               | ip protocol                                           | protocol              |
| proto         | d    | 6                 |                                                       | protocol              |
| state[l]      | s *  | ESTABLISHED       | protocol state for the "left-side state"              |                       |
| state[l]      | d    | 1                 |   (ie. the one from client to FortiGate)              |                       |
| state[r]      | s *  | ESTABLISHED       | protocol state for the "right-side state"             |                       |
| state[r]      | d    | 1                 |   (ie. the one from FortiGate to server)              |                       |
| duration      | d    | 70                | session duration                                      |                       |
| expire        | d    | 3529              | current session expiry timeout                        |                       |
| timeout       | d    | 3600              | configured session timeout                            |                       |
| sa            | s    | 100.50.20.10      | source IP address                                     |                       |
| da            | s    | 100.50.20.10      | destination IP address                                |                       |
| na            | s    | 10.20.30.40       | nat IP address (or 0.0.0.0 if not NAT is applied)     |                       |
| sp            | d    | 65324             | source port                                           |                       |
| dp            | d    | 53                | destination port                                      |                       |
| np            | d    | 43332             | nat port (or 0 if not NAT is applied)                 |                       |
| rate[u]       | s *  | 5.238 Mbps        | speed in upload [u] or download [d] direction         |                       |
| rate[d]       | s *  | 45.072 Kbps       |  in the most appropriate units (see [Rate section](/fortiformatter/output_format.md#rate))||
| rate[u]       | d    | 654807            | the same speed in Bytes/s with no units string,       |                       |
| rate[d]       | d    | 1195              |  upload direction is client to server                 |                       |
| rate[sum]     | d    | 1195              | the total speed of upload and download                |                       |
| count[op]     | d    | 3                 | number of packets forwarded on original direction     |                       |
| count[ob]     | d    | 390               | number of bytes forwarded on original direction       |                       |
| count[oe]     | d    | 1                 | number of errors forwarded on original direction      |                       |
| count[rp]     | d    | 1                 | number of packets forwarded on reverse direction      |                       |
| count[rb]     | d    | 60                | number of bytes forwarded on reverse direction        |                       |
| count[re]     | d    | 1                 | number of errors forwarded on reverse direction       |                       |
| npuflag[o]    | x    | 81                | NPU flag field for original direction                 |                       |
| npuflag[r]    | x    | 81                | NPU flag field for reverse direction                  |                       |
| offload[o]    | d    | 8                 | the NPU type for original direction offload           |                       |
| offload[r]    | s *  | Y or N            | is session offloaded to NPU in reverse direction?     |                       |
| offload[r]    | d    | 8                 | the NPU type for reverse direction offload            |                       |
| nturbo[o]     | s *  | Y or N            | is session nTurbo accelerated in original direction?  |                       |
| nturbo[o]     | d    | 0                 | the nTurbo type (usually 0 or 1) for original dir.    |                       |
| nturbo[r]     | s *  | Y or N            | is session nTurbo accelerated in reverse direction?   |                       |
| nturbo[r]     | d    | 0                 | the nTurbo type (usually 0 or 1) for reverse dir.     |                       |
| vdom          | d *  | 1                 | VDOM number for the session (see [VDOM section](/fortiformatter/output_format.md#vdom))||
| vdom          | s    | 1 or test         | string if modifier translates the number to VDOM name |                       |
| policy        | d *  | 10                | policy number (numbers can repeat in vdoms)           |                       |
| helper        | s    | dns-udp           | helper name if some was used (or "-")                 |                       |
| state         | s    | may_dirty,npu     | session states (see [Session state section](/fortiformatter/output_format.md#session-state))          |                       |
| haid          | d    | 0                 | HA ID                                                 |                       |
| plain         | s    | whole session     | the original string of the full session description   |                       |
| newline       | s    | \n                | replaced by the new line to create multiline outputs  |                       |
| innpu[o]      | s    | 12                | NPU ID the session packets are received by (or "-")   |                       |
| innpu[f]      | s    | 16                | NPU ID the session packets are forwared to (or "-")   |                       |
| outnpu[o]     | s    | 12                | NPU ID the session packets are sent to (or "-")       |                       |
| outnpu[f]     | s    | 16                | NPU ID the session packets are forwared to (or "-")   |                       |
| nooff[no]     | s    | dirty             | the exact value of "no_ofld_reason" field             |                       |
| nooff[ko]     | s    | not-established   | field "ofld_fail_reason", kernel, original direction  |                       |
| nooff[kr]     | s    | not-established   | field "ofld_fail_reason", kernel, reverse direction   |                       |
| nooff[do]     | s    | protocol-not...   | field "ofld_fail_reason", driver, original direction  |                       |
| nooff[dr]     | s    | protocol-not...   | field "ofld_fail_reason", driver, reverse direction   |                       |
| tunnel[i]     | s    | test              | IPSec tunnel name in incoming direction               |                       |
| tunnel[o]     | s    | test              | IPSec tunnel name in outgoing direction               |                       |
| shapingpolicy | s *  | 1                 | shaping policy id (or "-" when no shaping is done)    |                       |
| shapingpolicy | d    | 1                 | shaping policy id (or "0" when no shaping is done)    |                       |
| shaper[o]     | s    | shaperA           | Name of the shapper applied in original direction     |                       |
| shaper[r]     | s    | shaperB           | Name of the shapper applied in reverse direction      |                       |
| shaper[ip]    | s    | shaperC           | Name of the per-source-ip shaper                      |                       |
| mac[i]        | s    | 00:01:02:03:04:05 | Incoming ("source") MAC address                       | mac[src]              |
| mac[o]        | s    | 00:01:02:03:04:05 | Outgoing ("destination") MAC address                  | mac[dst]              |
| iface[oi]     | d    | 123               | Index if incoming interface in original direction     | iface[io]             |
| iface[oo]     | d    | 123               | Index if outgoing interface in original direction     |                       |
| iface[ri]     | d    | 123               | Index if incoming interface in reverse direction      | iface[ir]             |
| iface[ro]     | d    | 123               | Index if outgoing interface in reverse direction      | iface[or]             |
| nexthop[o]    | s    | 1.2.3.4           | IP address of next hop in original direction          | nh[o]                 |
| nexthop[r]    | s    | 1.2.3.4           | IP address of next hop in reverse direction           | nh[r]                 |
| user          | s    | user              | User name of the authenticated user or "-"            |                       |
| authserver    | s    | ourldap           | Profile name of the authentication server or "-"      |                       |
| authinfo      | d    | 3                 | Field "auth_info"                                     |                       |
| custom[...]   | s *  | whatever          | See [Custom fields section](/fortiformatter/output_format.md#custom-fields)  ||


Aliases have exactly the same meaning as the original name.

## Shortcuts

Some variables are very commonly used together with others, hence some shortcuts exist.
It is also a little more easy to format the output if exact width columns are needed.

|     name     | composed of                   | example output                                |
| ------------ | ----------------------------- | --------------------------------------------- |
| det          | duration/expire (timeout)     | 87/3513 (3600)                                |
| sap          | sa:sp                         | 1.2.3.4:43243                                 |
| dap          | da:dp                         | 8.8.8.8:53                                    |
| sdap         | sa:sp->da:dp                  | 1.2.3.4:43243->8.8.8.8:53                     |
| tunnels      | tunnel[i]->tunnel[o]          | -->Lab-Sophia                                 |
| patho        | iface[oi]->iface[oo] nh[o]    |  65->54    193.86.26.193                      |
| pathr        | iface[ri]->iface[ro] nh[r]    |  54->65   10.109.250.102                      |


## Macros

Macros are exactly like shortcuts from user's point of view, however internally they are processed
at the very begining and (unlike shortcuts) they can contain other variables and texts.
They are mainly use to simplify specification of the most common output formats.

|      name      | format string                                                                                                        |
| -------------- | -------------------------------------------------------------------------------------------------------------------- |
| default_basics | ${serial:08x}: ${vdom:3d}/${policy:-5s} ${protocol:-4s} ${state[l]:11s}/${state[r]:-11s} ${sap:-21s} -> ${dap:-21s}  |
| default_hw     | OFF(${offload[o]}/${offload[r]}), NTB(${nturbo[o]}/${nturbo[r]}) FLG(${npuflag[o]:#02x}/${npuflag[r]:#02x})          |
| default_rate   | RATE(up:${rate[u]:15s}, down:${rate[d]:15s})                                                                         |
| default_counts | COUNTS(org:${count[o]:15s}, rev:${count[r]:15s}))                                                                    |
| default_time   | TIME(${det:-20s})                                                                                                    |
| default_states | ${state:-60s|, }                                                                                                     |
| default_nooff  | ${nooff[ko]}/${nooff[kr]},${nooff[do]}/${nooff[dr]}                                                                  |
| default_path   | INF(org: ${patho}, rev: ${pathr})                                                                                    |
| default_macs   | ${mac[i]:17s} -> ${mac[o]:17s}                                                                                       |

Name `default_basic` refers to same format string as `default_basics` and `default_count` is the same as `default_counts`.

## Rate

Without any format string, the default rate format is `s`. String format always includes units and
these units can be different for each direction and also for each line. Numeric formats (`d`, `f`)
never contain any text.

Without any modifier the units are automaticaly selected. The possible modifiers are:

| units   | read as ...               | convert to bits |
| ------- | ------------------------- | --------------- |
| Kibps   | kibibits per second       | * 1024^1 bits   |
| Kbps    | kilobits per second       | * 1000^1 bits   |
| Mibps   | mebibits per second       | * 1024^2 bits   |
| Mbps    | megabits per second       | * 1000^2 bits   |
| Gibps   | gibibits per second       | * 1024^3 bits   |
| Gbps    | gigabits per second       | * 1000^3 bits   |
| Tibps   | tebibits per second       | * 1024^4 bits   |
| Tbps    | terabits per second       | * 1000^4 bits   |


Case does not matter except for `b` and `B`. The lowercase `b` means bits and the uppercase
`B` means bytes.

For example:

```
${rate[o]} / ${rate[r]}                                // 1.083 Mbps / 60.522 Mbps
${rate[o]|Bps} / ${rate[r]|Bps}                        // 135376.000 Bps / 7565278.000 Bps
${rate[o]:d|Bps} / ${rate[r]:d|Bps}                    // 135376 / 7565278
${rate[o]|Gibps} / ${rate[r]|Gibps}                    // 0.001 Gibps / 0.056 Gibps
${rate[o]:d|Gibps} / ${rate[r]:d|Gibps}                // 0 / 0
${rate[o]:10.8f|Gibps} / ${rate[r]:10.8f|Gibps}        // 0.00100863 / 0.05636571
```

## Vdom

Vdoms are shown as numbers by default in both `d` and `s` mode. However a modifier can
be specified to translate some numbers to names:

For example:

```
${vdom}                        // prints on new lines: 2, 1, 2, 0
${vdom:s}                      // prints on new lines: 2, 1, 2, 0
${vdom:s|1=something,2=else}   // prints on new lines: else, something, else, 0
```

## Session state

Shows all states divided by simple coma (without space). First modifier string specifies
the separator (which can have multiple characters).

Second modifier string can be composed of multiple fields divided by `;`. The field can be
`sort` to order the states by name or `filter:a,b,c` to only display states a, b and c
(if they are present).

For example:

```
${state}                            // log,may_dirty,npu,synced,f00,log-start
${state|, }                         // log, may_dirty, npu, synced, f00, log-start
${state|+|filter:log,f00,npu}       // log+npu+f00
${state|+|filter:log,f00,npu;sort}  // f00+log+npu
```

## Custom fields

Custom fields are completely independent on the result session fields (even though they can have the same name) and
are defined by plugins.

They are used to include some additional information (usually calculated by the plugin) there were not part of the original
session output.

These fields can be used similarly to the regular fields, however they need to be accessed via the `custom` field. For
example if you know that some plugin created custom field `myfield` (which should be written in the plugin documentation),
you can display it using `${custom|myfield}` formatter expression. 

To use a specific output format, use the standard formats after `custom` text. Like `${custom:d|myfield}`.
