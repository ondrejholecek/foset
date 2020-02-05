# SSH provider

As the Foset outputs are based on the saved commands outputs executed on FortiGate device, it is also possible to instruct Foset to collect the outputs directly over SSH without the need to execute the commands manually and save them.

## Basic info

- schema: `ssh://`
- rest: basically the command to run but it is slightly more complex, to make that easy to use without much thinking, following aliases are define (all are VDOMs aware so it is not necessary to think whether VDOM mode is enabled or not):
  - `ssh://sessions` - collect the output of `diagnose sys session list`
  - `ssh://vdoms` - collect the output of `diagnose sys vd list`
  - `ssh://interfaces` - collect the output of `diagnose netlink interface list`

## Connection info

SSH provider needs also the information how to connect to the FortiGate. For that the `-i ssh|...` parameter is used on Foset command line.

Following are the parameters recognized after `|`:
- `host` - FortiGate hostname or IP address
- `port` - Port SSH is running on (22 by default)
- `user` - Username to log in as ("admin" by default)
- `password` - Password for user (empty by default)
- `ask` - Used instead of `password` to request the password on standard input (with echo disabled)
- `agent` - Used instead of `password` or `ask` to use the SSH agent that must be already started
- `keepalive` - This is used when `--loop` (or `-l`) Foset parameter is used - i that case the SSH needs to stay connected and this option specifies how often to send some useless data (just "enter") over SSH connection to prevent it from timeouting. It is 45 seconds by default. Specify `0` to disable it. 

Note that `ssh` provider is not enabled when at least `host` parameter is not given and if `ssh://` schema is used in this case, the error will be `no provider found`.

## Examples

Connect to FortiGate running at 10.0.0.2 as user "ondrej" and use the SSH agent running on the computer:
`$ foset -i 'ssh|host=10.0.0.22,user=ondrej,agent' -r ssh://sessions`

Connect to 10.0.0.3 as default user "admin" with empty password
`$ foset -i 'ssh|host=10.0.0.3' -r ssh://sessions`

Connect to 10.0.0.4 as user "ondrej" and ask for password on standard input
```
$ foset -i 'ssh|host=10.0.0.4,user=ondrej,ask' -r ssh://sessions
Enter SSH password for ondrej@10.0.0.4 (port 22) :
[...]
```

Connect to 10.0.0.3 again, and this time also collect the information need to display the VDOM name:
```
$ foset -i 'ssh|host=10.0.0.3' -r ssh://sessions -p 'indexmap|vdoms=ssh://vdoms' \
  -o '${default_basic} ${custom|vdom}'
[...]
0000009e:   0/i     UDP         SEEN/SEEN        10.5.22.229:14977     -> 10.5.31.255:8014      root
[...]
```

The same as above, but collect also interface info:
```
$ foset -i 'ssh|host=10.0.0.3' -r ssh://sessions \
  -p 'indexmap|vdoms=ssh://vdoms,interfaces=ssh://interfaces' \
  -o '${default_basic} ${custom|vdom} ${custom|iface[io]}'
[...]
00abbaf3:   1/i     UDP         SEEN/SEEN        20.20.20.2:52970      -> 20.20.20.1:3784       LTE VLAN123
[...]
```

## Full schema specification

When not using the three aliases above, the full path has format "`ssh://`*vdom*`/`*cmdtype*`/`*command*", where:
- `vdom` (only relevant when VDOM mode is enabled, but must be specified all the times) is either the exact name of the VDOM (note that case matters) or one of the keywords:
  - `<global>` to run in global context
  - `<mgmt>` to run in current management VDOM
- `cmdtype` specifies how to collect the output:
  - `simple` - just run the command and wait until it completes (this is good for simple and fast commands like vdom list or interface list)
  - `long` - this is for command running longer time (like session list) where the output is transparently collected on the background and Foset can continue with parsing it on "realtime" where it gets only part of the output at the time
- `command` is the actual command to run which space replaced by `/` (so `diagnose sys session list` is written as `diagnose/sys/session/list`)

The aliases above written as real paths:
- `ssh://vdoms` = `<global>/simple/diagnose/sys/vd/list`
- `ssh://interfaces` = `<mgmt>/simple/diagnose/netlink/interface/list`
- `ssh://sessions` = `<mgmt>/long/diagnose/sys/session/list`
