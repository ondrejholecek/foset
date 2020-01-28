# CONDITIONS (FILTER) FORMAT

Filter is composed of one condition or multiple conditions separated by "and"
or "or" keywords. "Or" has precedence over "and", if that is unwanted, parenthesis
can be used to specify the precedence manually.

One condition can have any number of words, but one, two or three words are usually
used. First word always refers to the "session field" (ie. some parameter of the session
that was parsed from the FortiGate session output).

If there are three or more words, the second is always operator and third and all others
compose the "compare to" string.


If there are only two words, the second one is the "compare to" string and the operator
is implicit and depends on the session field that is being checked.

If only the session field is present, it usually means that this field must be present/enabled/possitive
(etc. - depending on the session field being checked).

Quatation marks have no special meaning.

## All supported session fields

Check the operator group tables bellow to find what comparisons are possible.



|     name      | example value     | operator group     | description                                    | alias          |
| ------------- | ----------------- | ------------------ | ---------------------------------------------- | -------------- |
| host          | 1.2.3.4           | ip-match           | source or destination or natted IP address     | -              |
| shost         | 1.2.3.4           | ip-match           | source host IP address                         | -              |
| dhost         | 1.2.3.4           | ip-match           | destination host IP address                    | -              |
| nhost         | 1.2.3.4           | ip-match           | natted IP address                              | -              |
| port          | 53                | number-match       | source or destination or nat port              | -              |
| sport         | 65342             | number-match       | source port                                    | -              |
| dport         | 80                | number-match       | destination port                               | -              |
| nport         | 33440             | number-match       | natted port                                    | -              |
| policy        | 10                | number-match (\*1  | policy number                                  | -              |
| vdom          | 1                 | number-match       | vdom number                                    | -              |
| helper        | dns-udp           | string-match       | helper name                                    | -              |
| state         | log-start         | string-match (\*2) | at least one of the states match               | -              |
| status[l]     | 1                 | number-match (\*3) | client to FortiGate (left) session status      | -              |
| status[r]     | 4                 | number-match (\*3) | FortiGate to server (right) session status     | -              |
| status        | established       | number-match (\*3) | either client-to-FGT or FGT-to-client s.s.     | -              |
| proto         | 6                 | number-match (\*4) | IP protocol number                             | protocol       |
| serial        | 0xea1fa572        | number-match       | Session serial number                          | session        |
| npuflag[o]    | 0x81              | number-match       | NPU flag for original direction                |                |
| npuflag[r]    | 0x81              | number-match       | NPU flag for reverse direction                 |                |
| npuflag       | 0x81              | number-match       | NPU flag for original and reverse direction    |                |
| offload[o]    | 8                 | number-match       | NPU offload for original direction             | offloaded[o]   |
| offload[r]    | 8                 | number-match       | NPU offload for reverse direction              | offloaded[r]   |
| offload       | 8                 | number-match       | NPU offload for reverse and original direction | offloaded      |
| nturbo[o]     | 1                 | number-match       | NPU nturbo for original direction              | -              |
| nturbo[r]     | 1                 | number-match       | NPU nturbo for reverse direction               | -              |
| nturbo        | 1                 | number-match       | NPU nturbo for reverse and original direction  | -              |
| noooff[no]    | dirty             | string-match       | Field "no_ofld_reason"                         | -              |
| noooff[ko]    | none              | string-match       | Field "ofld_fail_reason": kernel, original     | -              |
| noooff[kr]    | not-established   | string-match       | Field "ofld_fail_reason": kernel, reverse      | -              |
| noooff[do]    | none(0)           | string-match       | Field "ofld_fail_reason": driver, original     | -              |
| noooff[dr]    | none(0)           | string-match       | Field "ofld_fail_reason": driver, reverse      | -              |
| innpu[o]      | 20                | number-match       | NPU ID of the original incoming NPU            | -              |
| innpu[f]      | 21                | number-match       | NPU ID of the forwarded to incoming NPU        | -              |
| outnpu[o]     | 20                | number-match       | NPU ID of the original outgoing NPU            | -              |
| outnpu[f]     | 21                | number-match       | NPU ID of the forwarded to outgoing NPU        | -              |
| rate[u]       | 10 Mbps           | rate-match         | Data rate in upload direction (original)       | upload         |
| rate[d]       | 1 mBps            | rate-match         | Data rate in download direction (reverse)      | download       |
| rate          | 100 kBps          | rate-match         | Data rate in either upload or download dir.    | -              |
| rate[sum]     | 1 Gbps            | rate-match         | Summary data rate in both directions           | -              |
| count[ob]     | 10000             | number-match       | Statistics: bytes forwarded in original dir.   | stats[ob]      |
| count[op]     | 10                | number-match       | Statistics: packets forwarded in original dir. | stats[op]      |
| count[oe]     | 1                 | number-match       | Statistics: errors forwarded in original dir.  | stats[oe]      |
| count[rb]     | 10000             | number-match       | Statistics: bytes forwarded in reverse dir.    | stats[rb]      |
| count[rp]     | 10                | number-match       | Statistics: packets forwarded in reverse dir.  | stats[rp]      |
| count[re]     | 1                 | number-match       | Statistics: errors forwarded in reverse dir.   | stats[re]      |
| tunnel[i]     | test              | string-match       | Name if IPSec tunnel session came from         | -              |
| tunnel[o]     | test              | string-match       | Name if IPSec tunnel session goes to           | -              |
| tunnel        | test              | string-match       | Name if either incoming or outgoing IPSec      | -              |
| shapingpolicy | 1                 | number-match       | Shaping policy id                              | -              |
| shaper[o]     | shaperA           | string-match       | Name if shaper applied in original direction   | -              |
| shaper[r]     | shaperB           | string-match       | Name if shaper applied in reverse direction    | -              |
| shaper[ip]    | shaperC           | string-match       | Name if per-source-ip shaper                   | shaper[pip]    |
| shaper        | shaperD           | string-match       | Name if any type of shaper applied             | -              |
| mac[i]        | 00:01:02:03:04:05 | string-match       | Incoming ("source") MAC address                | mac[src] smac  |
| mac[o]        | 00:01:02:03:04:05 | string-match       | Outgoing ("destination") MAC address           | mac[dst] dmac  |
| mac           | 00:01:02:03:04:05 | string-match       | Either incoming or outgoing MAC address        | -              |
| iface[oi]     | 102               | number-match       | Incoming interface in original direction       | iface[io]      |
| iface[oo]     | 103               | number-match       | Outgoing interface in original direction       | -              |
| iface[ri]     | 103               | number-match       | Incoming interface in reverse direction        | iface[ir]      |
| iface[ro]     | 102               | number-match       | Outgoing interface in reverse direction        | iface[or]      |
| iface         | 102               | number-match       | Incoming or outgoing interface in any dir.     | -              |
| nexthop[o]    | 1.2.3.4           | ip-match           | Next hop in original direction                 | nh[o]          |
| nexthop[r]    | 1.2.3.4           | ip-match           | Next hop in reverse direction                  | nh[r]          |
| nexthop       | 1.2.3.4           | ip-match           | Next hop in any direction                      | -              |
| user          | someuser          | string-match       | User name of authenticated user                | -              |
| authserver    | ourldap           | string-match       | Auth profile nane                              | -              |
| authinfo      | 3                 | number-match       | Auth info                                      | -              |
| custom        |                   | string-match       | Special field, see [Custom match section](/forticonditioner/README.md#custom-match) | - |


- (\*1) Policy can also be string "internal"
- (\*2) Since each state is evaluated individually, negation has to follow the form "not state ..." instead of "state not ..."
- (\*3) This is mached by session state number, but following texts are translated to numbers automatically.

| string starts ... | full state name | session state number | aliases       | comment                     |
| ----------------- | --------------- | -------------------- | ------------- | --------------------------- |
| unseen            | unseen          | 0                    | -             | UDP but also matches TCP    |
| seen              | seen            | 1                    | -             | UDP but also matches TCP    |
| n                 | none            | 0                    | -             | TCP but also matches UDP    |
| e                 | established     | 1                    | -             | TCP but also matches UDP    |
| syn_s             | syn_sent        | 2                    | ss syn-s      |                             |
| syn_r             | syn_recv        | 3                    | sr syn-r      |                             |
| fin_wait1         | fin_wait1       | 4                    | fw1 fin-wait1 |                             |
| fin_wait2         | fin_wait2       | 5                    | fw2 fin-wait2 |                             |
| t                 | time_wait       | 6                    | -             |                             |
| close             | close           | 7                    | -             |                             |
| close_            | close_wait      | 8                    | cw close-w    |                             |
| la                | last_act        | 9                    | -             |                             |
| li                | listen          | 10                   | -             |                             |
| closi             | closing         | 11                   | -             |                             |


- (\*4) This is mached by IP protocol number, but following texts are translated to numbers automatically.

| text        | protocol |
| ----------- | -------- |
| icmp        | 1        |
| tcp         | 6        |
| udp         | 17       |
| esp         | 50       |


## Operators

Some operator groups have their own unequal sign. Regardless, the negation can be always done
by prefixing `!` to the operator (it has to form one word). Another way to negate is to start
the condition with either "not " or "no "

Condition can also start with "has " or "is " which have no special meaning and can just be used
to make better looking expressions.

### Operator group: ip-match

For matching IP address. Only IPv4 supported at this moment. Be careful when using negation with
variable like "host", because the negation means that either source or destination address does not
match - which is likely to happen in most cases.


| operators           | meaning                           | right side             | is default |
|-------------------- | --------------------------------- | ---------------------- | ---------- |
| =  ==  is           | IP exactly equal                  | IP address             | yes        |
| !=  <>  not         | IP differs                        | IP address             |            |
| in                  | IP is in the subnet               | CIDR (like 10.0.0.0/8) |            |


### Operator group: number-match

For matching integers of any length. "Compare to" string can start with "0x" to denote hex number.

| operators           | meaning                           | right side             | is default |
|-------------------- | --------------------------------- | ---------------------- | ---------- |
| =  ==  eq  is       | numbers exactly equal             | decimal or hex number  | yes        |
| <>  ne  not  !=     | numbers not equal                 | decimal or hex number  |            |
| >  gt               | session field is greater than     | decimal or hex number  |            |
| >=  ge              | session field is greater or equal | decimal or hex number  |            |
| <  lt               | session field is lesser than      | decimal or hex number  |            |
| <=  le              | session field is lesser or equal  | decimal or hex number  |            |



### Operator group: string-match

For matching strings. By default comparing is case sensitive, but with the operator starts with `#`,
the case is ignored.

| operators               | meaning                           | right side             | is default |
| ----------------------- | --------------------------------- | ---------------------- | ---------- |
| =  ==  eq  is           | strings exactly the same          | string (do not use "") | yes        |
| <>  ne  not !=          | strings differ                    | string (do not use "") |            |
| prefix starts start     | string on left starts with        | string (do not use "") |            |
| suffix ends end         | string on left ends with          | string (do not use "") |            |
| contain contains c has  | string on left contains           | string (do not use "") |            |


### Operator group: rate-match

For matching data rates using various supported units.
The operators are the same as with number-match group, except the possiblity to use hex prefix.
Case does not matter except for `b` and `B`. The lowercase `b` means bits and the uppercase `B` means bytes.
```
|---------+---------------------------+-----------------|
| units   | read as ...               | convert to bits |
|---------+---------------------------+-----------------|
| Kibps   | kibibits per second       | * 1024^1 bits   |
| Kbps    | kilobits per second       | * 1000^1 bits   |
| Mibps   | mebibits per second       | * 1024^2 bits   |
| Mbps    | megabits per second       | * 1000^2 bits   |
| Gibps   | gibibits per second       | * 1024^3 bits   |
| Gbps    | gigabits per second       | * 1000^3 bits   |
| Tibps   | tebibits per second       | * 1024^4 bits   |
| Tbps    | terabits per second       | * 1000^4 bits   |
|---------+---------------------------+-----------------|
```

## Custom match

Custom fields are fields not created by parsing the session dump, but rather created externally usually via some plugins. 
These fields have their type specified individually, hence the right operator group depends on the specific field. 

In the filter expression, the name of the custom field must be preceeded by the string `custom`. This is to correctly 
identify the field, because the name of the custom field can be the same as the name of regular field. For example,
to filter on the VDOM name custom field created by a plugin, following expression must be used: `custom vdom root`. Without
the word `custom` you would get an error because the regular `vdom` field is integer and string `root` cannot be
converted to integer.

## Match two fields agains each other

Normally left side of the filter expression is the field name and the right side is its value.

Using another's field name on the right side is not really supported, however there is an experimental feature "Nested 
formatter" to dynamically create the right side using an independent [formatter expression](/fortiformatter). 
It is completely independent on the main output formatter expression defined with `-o`. Such expression can be only 
on the right side and must be fully surrounded by the `|` characters. 

When evaluating the filter expression, the formatter expression is evaluated first and its result is used on the right
side in the same way as if the user wrote it there directly.

### example

The requirement is to match the sessions that use the same source and destionation port number.

Following filter does not work because regular filter fields cannot appear on the right side:

```
$ foset -r ~/tmp/core -f 'sport = dport'
2020-01-28 22:47:08 CRITICAL foset.forticonditioner conditioner.go:1187 Generic check "sport" error: strconv.ParseUint: parsing "dport": invalid syntax
```

Following filter uses nester formatter to generate the right side and it fulfils the requirement:

```
$ foset -r ~/tmp/core -f 'sport = |${dp}|'

[...]
57528992:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:123       -> 217.30.75.147:123
68ff91ee:   0/11    UDP         SEEN/UNSEEN      10.109.19.23:123      -> 31.31.74.35:123
68ffb0f8:   0/11    UDP         SEEN/UNSEEN      10.109.51.206:123     -> 89.221.218.101:123
[...]
```
