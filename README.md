# FOrtigate SEssion Tool

This command line tool utilizes the [FortiSession library](https://github.com/ondrejholecek/fortisession) to parse the text output of `diagnose sys session list` command.

It can read either plain-text files (Putty log output or Linux "script" command output) specified with `-f` or `--file` command line parameter, or the same files compressed with Gzip (add also `-g`).

The session fields can be displayed in different formats controlled by "format string" given as command line parameter (`-o` or `--output`). To learn how to specify the output, see [Output format description](https://github.com/ondrejholecek/fortisession/blob/master/fortiformatter/output_format.md).

All sessions can be displayed or only interesting sessions can be selected using the "filter string" given as another command line parameters (`-f` or `--filter`). To learn how to use the conditions to select the right sessions, see [Condition format description](https://github.com/ondrejholecek/fortisession/blob/master/forticonditioner/condition_format.md).

## Download

Pre-built binaries are available for the most commonly used operating systems. Those are single executable files without any dependencies, that can be executed without any installation.

- [Windows (64bit)](https://github.com/ondrejholecek/foset/raw/master/release/1.4/windows/foset.exe)
- [Linux (64bit)](https://github.com/ondrejholecek/foset/raw/master/release/1.4/linux/foset)
- [MacOS (64bit)](https://github.com/ondrejholecek/foset/raw/master/release/1.4/macos/foset)

## Examples

**Note: If neither `-r` nor `--file` options are specified, Foset will try to read the plain-text sessions from standard output. Usually this is not what you want.**

### default output from compressed session file

If not specified otherwise, the default output format is `${default_basic} ${default_hw}, ${default_rate}, ${default_counts}`:

*NOTE: The second column is VDOM/policy IDs.*

```
$ foset -r /tmp/sessions.gz -g

68fb0e9e:   0/4     TCP         NONE/ESTABLISHED 172.26.81.24:4125     -> 10.109.19.170:22      OFF(N/Y), NTB(N/N) FLG(0x00/0x81), RATE(up:      8.000 bps, down:     16.000 bps), COUNTS(org:      5228/89/1, rev:     8348/123/1)
68ff7423:   0/39    UDP         SEEN/UNSEEN      10.109.250.115:4251   -> 208.91.112.52:53      OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:     40.000 bps, down:    264.000 bps), COUNTS(org:      2024/32/1, rev:     13082/32/1)
043e7840:   0/0     UDP         SEEN/UNSEEN      10.109.248.1:5246     -> 10.109.248.2:5246     OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:    112.000 bps, down:    152.000 bps), COUNTS(org:167928679/1339136/1, rev:211927469/1339153/1)
68ffa62f:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.194:53    OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      8.000 bps, down:      0.000 bps), COUNTS(org:        100/1/1, rev:         72/1/1)
68ffa631:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.195:53    OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      8.000 bps, down:      0.000 bps), COUNTS(org:        100/1/1, rev:         72/1/1)
68ff7e54:   0/30    UDP         SEEN/UNSEEN      10.109.16.23:42228    -> 10.109.80.54:61904    OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      0.000 bps, down:      0.000 bps), COUNTS(org:        944/2/1, rev:         64/2/1)
68ffa630:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.196:53    OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      8.000 bps, down:      0.000 bps), COUNTS(org:        100/1/1, rev:         72/1/1)
68ff1e63:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:62521  -> 208.91.112.52:53      OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      0.000 bps, down:      0.000 bps), COUNTS(org:      5970/95/1, rev:     40837/95/1)
68ffa62e:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.198:53    OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      8.000 bps, down:      0.000 bps), COUNTS(org:        100/1/1, rev:         72/1/1)
68ff9482:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:58712     -> 199.249.120.1:53      OFF(N/N), NTB(N/N) FLG(0x00/0x00), RATE(up:      0.000 bps, down:      0.000 bps), COUNTS(org:         78/1/1, rev:        833/1/1)
```

### basic information only

To create some specific filter one can start with only `${default_basic}` filter, which really gives only the basic information about the session: serial number, VDOM/policy, protocol, states, IPs and ports.

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic}'

68fb0e9e:   0/4     TCP         NONE/ESTABLISHED 172.26.81.24:4125     -> 10.109.19.170:22
043e7840:   0/0     UDP         SEEN/UNSEEN      10.109.248.1:5246     -> 10.109.248.2:5246
68ff7423:   0/39    UDP         SEEN/UNSEEN      10.109.250.115:4251   -> 208.91.112.52:53
68ffa62f:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.194:53
68ff7e54:   0/30    UDP         SEEN/UNSEEN      10.109.16.23:42228    -> 10.109.80.54:61904
68ffa631:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.195:53
68ff1e63:   0/39    UDP         SEEN/UNSEEN      10.109.250.110:62521  -> 208.91.112.52:53
68ffa630:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.196:53
68ff9482:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:58712     -> 199.249.120.1:53
68ffa62e:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.198:53
```

### add NAT IP address and port

If NAT IP address and port are important, those can be added:

*Note: instead of `${na}:${np}` a shortcut `${nap}` can be used with the same result.*

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${na}:${np}'

68fb0e9e:   0/4     TCP         NONE/ESTABLISHED 172.26.81.24:4125     -> 10.109.19.170:22      0.0.0.0:0
68ff7423:   0/39    UDP         SEEN/UNSEEN      10.109.250.115:4251   -> 208.91.112.52:53      193.86.26.196:64667
68ffa62f:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.194:53    193.86.26.196:41526
68ffa631:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.195:53    193.86.26.196:41526
68ffa630:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.196:53    193.86.26.196:41526
68ffa62e:   0/1     UDP         SEEN/UNSEEN      10.109.3.9:41526      -> 173.243.138.198:53    193.86.26.196:41526
68ffa881:   0/1     UDP         SEEN/UNSEEN      10.109.3.36:38816     -> 8.8.8.8:53            193.86.26.196:38816
68ffadd8:   0/39    UDP         SEEN/UNSEEN      10.109.250.111:19284  -> 65.210.95.239:53      193.86.26.196:19284
68f41e9f:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49244   -> 81.0.212.201:80       0.0.0.0:0
684fdb59:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49180   -> 81.0.212.201:80       0.0.0.0:0
```

It can be seen seen that some sessions are not NATted.

### find more unNATted sessions

We can use also filter to search for NAT IP 0.0.0.0, which signalizes no NAT. IP filters are those of few ones that have different keyword in filter and in output format string:

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${nap}' -f 'nhost 0.0.0.0'

68fb0e9e:   0/4     TCP         NONE/ESTABLISHED 172.26.81.24:4125     -> 10.109.19.170:22      0.0.0.0:0
043e7840:   0/0     UDP         SEEN/UNSEEN      10.109.248.1:5246     -> 10.109.248.2:5246     0.0.0.0:0
68f41e9f:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49244   -> 81.0.212.201:80       0.0.0.0:0
684fdb59:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49180   -> 81.0.212.201:80       0.0.0.0:0
4ebfaff4:   0/i     UDP         SEEN/SEEN        10.109.19.67:15717    -> 10.109.31.255:8014    0.0.0.0:0
5defd5d8:   0/0     TCP         NONE/ESTABLISHED 10.109.3.254:11435    -> 10.109.3.8:514        0.0.0.0:0
68ffac92:   0/21    UDP         SEEN/UNSEEN      193.86.26.197:39223   -> 208.91.113.75:53      0.0.0.0:0
68ffa366:   0/12    UDP         SEEN/UNSEEN      10.109.20.28:12802    -> 10.109.3.14:53        0.0.0.0:0
68ff2f25:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49181   -> 81.0.212.203:80       0.0.0.0:0
6841b59d:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49181   -> 81.0.212.201:80       0.0.0.0:0
```

There are both UDP and TCP sessions matching.

### focus on TCP sessions

We can enhance the previous filter to get only TCP sessions:

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${nap}' -f 'nhost 0.0.0.0 and proto tcp'
68fb0e9e:   0/4     TCP         NONE/ESTABLISHED 172.26.81.24:4125     -> 10.109.19.170:22      0.0.0.0:0
68f41e9f:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49244   -> 81.0.212.201:80       0.0.0.0:0
684fdb59:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:49180   -> 81.0.212.201:80       0.0.0.0:0
5defd5d8:   0/0     TCP         NONE/ESTABLISHED 10.109.3.254:11435    -> 10.109.3.8:514        0.0.0.0:0
68ffb6e9:   0/13    TCP         NONE/ESTABLISHED 172.26.52.48:52957    -> 10.109.3.18:443       0.0.0.0:0
68f48f49:   0/21    TCP  ESTABLISHED/ESTABLISHED 193.86.26.197:19967   -> 81.0.212.202:80       0.0.0.0:0
6099b7b6:   0/49    TCP         NONE/ESTABLISHED 10.109.248.18:37120   -> 10.109.3.42:443       0.0.0.0:0
68fdad79:   0/16    TCP         NONE/ESTABLISHED 172.26.48.47:57021    -> 10.109.3.7:443        0.0.0.0:0
68fdad78:   0/16    TCP         NONE/ESTABLISHED 172.26.48.47:57020    -> 10.109.3.7:443        0.0.0.0:0
68ffb713:   0/4     TCP         NONE/ESTABLISHED 172.26.48.47:63579    -> 10.109.16.84:443      0.0.0.0:0
```

### focus on firewall policy id

We are futher interested only in session allowed by firewall policy 4 in root VDOM (0 in this case):

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${nap}' -f 'nhost 0.0.0.0 and proto tcp and policy 4 and vdom 0'

68fb0e9e:   0/4     TCP         NONE/ESTABLISHED 172.26.81.24:4125     -> 10.109.19.170:22      0.0.0.0:0
68ffb713:   0/4     TCP         NONE/ESTABLISHED 172.26.48.47:63579    -> 10.109.16.84:443      0.0.0.0:0
68ffb710:   0/4     TCP         NONE/ESTABLISHED 172.26.48.47:63578    -> 10.109.16.84:443      0.0.0.0:0
68fd718b:   0/4     TCP         NONE/ESTABLISHED 172.26.48.23:63310    -> 10.109.20.11:23       0.0.0.0:0
68ffb714:   0/4     TCP         NONE/ESTABLISHED 172.26.48.47:63580    -> 10.109.16.84:443      0.0.0.0:0
680b41be:   0/4     TCP         NONE/ESTABLISHED 172.26.48.18:62098    -> 10.109.19.184:22      0.0.0.0:0
68ffb5d2:   0/4     TCP         NONE/ESTABLISHED 172.26.48.7:52212     -> 10.109.19.227:80      0.0.0.0:0
68ffb528:   0/4     TCP         NONE/ESTABLISHED 172.26.48.58:8470     -> 10.109.16.20:80       0.0.0.0:0
68ffb641:   0/4     TCP         NONE/ESTABLISHED 172.26.48.47:63573    -> 10.109.16.142:80      0.0.0.0:0
68ffb664:   0/4     TCP         NONE/FIN_WAIT2   172.26.48.7:52215     -> 10.109.19.227:80      0.0.0.0:0
```

And let's check only those that are in time-wait state:

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${nap}' -f 'nhost 0.0.0.0 and proto tcp and policy 4 and vdom 0 and status time-wait'

68ffb67c:   0/4     TCP         NONE/TIME_WAIT   172.26.48.9:53810     -> 10.109.16.30:443      0.0.0.0:0
68ffb66c:   0/4     TCP         NONE/TIME_WAIT   172.26.48.9:53809     -> 10.109.16.220:443     0.0.0.0:0
```

### show all sessions with firewall authentication

*Note: here the filter `has user` was used. Although accepted, `has` doesn't have any real meaning and using the filter `user` would give exactly the same result, because that shows sessions where user field is not empty. Many other fields can be used with the same meaning.*

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic}' -f 'has user'

68ffb6e9:   0/13    TCP         NONE/ESTABLISHED 172.26.52.48:52957    -> 10.109.3.18:443
68ffb6b3:   0/4     TCP         NONE/SYN_SENT    172.26.52.48:52956    -> 10.108.201.2:6690
68ffb602:   0/4     TCP         NONE/SYN_SENT    172.26.52.48:52951    -> 10.108.201.2:6690
```

We can see the authenticated sessions but we don't know the user name. Let's modify the output string to show it as well, together with authentication server that was this user authenticated against:

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${user} (${authserver})' -f 'has user'

68ffb6e9:   0/13    TCP         NONE/ESTABLISHED 172.26.52.48:52957    -> 10.109.3.18:443       oholecek (FTNT-Corporate)
68ffb6b3:   0/4     TCP         NONE/SYN_SENT    172.26.52.48:52956    -> 10.108.201.2:6690     oholecek (FTNT-Corporate)
68ffb602:   0/4     TCP         NONE/SYN_SENT    172.26.52.48:52951    -> 10.108.201.2:6690     oholecek (FTNT-Corporate)
```

### session serial only

The output string `${default_basic}` is actually composed of many simple fields: `${serial:08x}: ${vdom:3d}/${policy:-5s} ${proto:-4s} ${state[l]:11s}/${state[r]:-11s} ${sap:-21s} -> ${dap:-21s}`. We can of course use the individual fields directly:

```
$ foset -r /tmp/sessions.gz -g -o '${sa} ${serial}'

172.26.81.24 68fb0e9e
10.109.248.1 43e7840
10.109.16.23 68ff7e54
10.109.250.110 68ff1e63
10.109.3.14 68ff9482
10.109.16.220 68ff8d58
10.109.16.220 68ff8e43
10.109.16.178 68ffb6cf
10.109.20.19 68fbb16a
10.109.20.19 68fbb168
```

And if some alignment is desired, we can specify the width of the source IP address field:

```
$ foset -r /tmp/sessions.gz -g -o '${sa:20s} ${serial}'

        172.26.81.24 68fb0e9e
        10.109.248.1 43e7840
        10.109.16.23 68ff7e54
      10.109.250.110 68ff1e63
         10.109.3.14 68ff9482
      10.109.250.115 68ff7423
       10.109.16.220 68ff8d58
          10.109.3.9 68ffa62f
          10.109.3.9 68ffa631
          10.109.3.9 68ffa630
```

Alignment to the left side is much nicer in this case:

```
$ foset -r /tmp/sessions.gz -g -o '${sa:-20s} ${serial}'

172.26.81.24         68fb0e9e
10.109.248.1         43e7840
10.109.16.23         68ff7e54
10.109.250.110       68ff1e63
10.109.3.14          68ff9482
10.109.16.220        68ff8d58
10.109.16.220        68ff8e43
10.109.16.178        68ffb6cf
10.109.20.19         68fbb16a
10.109.20.19         68fbb168
```

And also make sure that the hexadecimal session serial number starts with zero(s) if it is shorter than 8 characters (like the second session above and bellow):

```
$ foset -r /tmp/sessions.gz -g -o '${sa:-20s} ${serial:08x}'

172.26.81.24         68fb0e9e
10.109.248.1         043e7840
10.109.16.23         68ff7e54
10.109.250.110       68ff1e63
10.109.3.14          68ff9482
10.109.16.220        68ff8d58
10.109.250.105       68ffa815
10.109.16.173        68fbafad
10.109.16.220        68ff8e43
10.109.250.115       68ff7423
```

### downloading at least at some rate

If we are interested in sessions where somebody is downloading faster than 1 megabit per second:

*Note: keep in mind that Foset can only use the information present in the session list output - if the session if offloaded to NPU and it is not reporting the speed, this output will not be accurate.*

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic} ${default_rate})' -f 'rate[d] >= 1 Mbps'

68ffb513:   0/i     TCP         NONE/ESTABLISHED 193.85.189.20:52946   -> 193.86.26.196:22      RATE(up:    43.120 Kbps, down:     1.540 Mbps))
```

The `i` instead of firewall policy ID means that the session is internal - created by FortiGate and not matching any configured policy. Foset shows `i` for simplification, but we can get the real number by using the `${policy}` with default decimal format (`${default_basic}` macro uses string `{policy:-5s}`).

Let's also display the download speed without units and in kilobytes per second - we use the `${rate[d]}` field directly, change format to decimal (`${rate[d]:d}`) from default string and say we want to have this is kBps (`${rate[d]:d|kBps`). 

```
$ foset -r /tmp/sessions.gz -g -o '${serial} ${policy} ${sdap} Download:${rate[d]:d|kBps}' -f 'rate[d] >= 1 Mbps'
68ffb513 4294967295 193.85.189.20:52946->193.86.26.196:22 Download:193
```

Be careful to use the uppercase `B` for bytes and not the lowercase `b` for bits:

```
$ foset -r /tmp/sessions.gz -g -o '${serial} ${policy} ${sdap} Download:${rate[d]:d|kbps}' -f 'rate[d] >= 1 Mbps'

68ffb513 4294967295 193.85.189.20:52946->193.86.26.196:22 Download:1540
```

If better precision is needed, float format can be used:

```
$ foset -r /tmp/sessions.gz -g -o '${serial} ${policy} ${sdap} Download:${rate[d]:6.2f|kbps}' -f 'rate[d] >= 1 Mbps'

68ffb513 4294967295 193.85.189.20:52946->193.86.26.196:22 Download:1540.44
```

### session state flags

Field `${state}` is a little special because it can contain many state flags. By default they are only delimited by coma:

```
$ foset -r /tmp/sessions.gz -g -o '${serial:08x} ${state}'

68fb0e9e may_dirty,npu
043e7840 log,local,nds
68ff7e54 npu
68ff1e63 dirty,may_dirty,npu
68ff9482 dirty,may_dirty,npu
68ff8d58 log,dirty,may_dirty,per_ip,npu,f00
68ff8e43 log,dirty,may_dirty,per_ip,npu,f00
68ffb6cf log,may_dirty,per_ip,npu,nlb,f00,app_valid
68fbb16a log,may_dirty,per_ip,npu,f00
68fbb168 log,may_dirty,per_ip,npu,f00
```

But we can use the modifier field to select different delimiter:

```
$ foset -r /tmp/sessions.gz -g -o '${serial:08x} ${state|, }'

68fb0e9e may_dirty, npu
043e7840 log, local, nds
68ff7e54 npu
68ff1e63 dirty, may_dirty, npu
68ff9482 dirty, may_dirty, npu
68ff8d58 log, dirty, may_dirty, per_ip, npu, f00
68ff8e43 log, dirty, may_dirty, per_ip, npu, f00
68ffb6cf log, may_dirty, per_ip, npu, nlb, f00, app_valid
68fbb16a log, may_dirty, per_ip, npu, f00
68fbb168 log, may_dirty, per_ip, npu, f00
```

And we may also say that we are interested only in some flags - so that the other flags are not displayed even though they are present in the session state field:

```
$ foset -r /tmp/sessions.gz -g -o '${serial:08x} ${state|, |filter:npu,per_ip}'

68fb0e9e npu
043e7840
68ff7e54 npu
68ff1e63 npu
68ff9482 npu
68ff8d58 per_ip, npu
68ff8e43 per_ip, npu
68ffb6cf per_ip, npu
68fbb16a per_ip, npu
68fbb168 per_ip, npu
```

This of course does not prevent us from filtering on non-displayed state flags:

```
$ foset -r /tmp/sessions.gz -g -o '${serial:08x} ${state|, |filter:npu,per_ip}' -f 'state has app_valid'

68ffb6cf per_ip, npu
68ffaf94 per_ip, npu
68ff9577 per_ip, npu
68ffac92 npu
68ffb5d9 npu
68ffaed4 per_ip, npu
68fbed61 per_ip, npu
68ffb176 per_ip, npu
68d474ce per_ip, npu
6810987a per_ip, npu
```

### IP addresses within certain range

In addition to exact IP matching, we can also show sessions where some IP address is within certain specified range:

```
$ foset -r /tmp/sessions.gz -g -o '${default_basic}' -f 'dhost in 2.22.0.0/16'

68ffa67c:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:40648     -> 2.22.230.129:53
68ffa6da:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:35303     -> 2.22.11.30:53
68ff930b:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:50768     -> 2.22.10.171:53
68ffa67d:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:47151     -> 2.22.230.129:53
68ff9ef7:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:44271     -> 2.22.11.21:53
68ffaaad:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:46162     -> 2.22.11.37:53
68ffa8a5:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:37676     -> 2.22.230.130:53
68ffae8a:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:47766     -> 2.22.11.93:53
68ffa679:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:59927     -> 2.22.230.129:53
68ffa67b:   0/1     UDP         SEEN/UNSEEN      10.109.3.14:36538     -> 2.22.230.129:53
```

## Further processing

Foset itself does not have any capability for sorting the outputs or creating any aggregated statistics. It is expected that other tools are used for that. 

Like in the following example where we are showing to 10 sessions sorted by download speed from fastest to slowest:

```
$ foset -r /tmp/sessions.gz -g -o '${rate[d]:010d|bps} ${serial:08x} ${sdap:-45s} ${rate[d]}' | sort -n -r | head -10

0001540440 68ffb513 193.85.189.20:52946->193.86.26.196:22         1.540 Mbps
0000124960 68ffb5b6 10.109.19.73:34520->209.222.147.40:443        124.960 Kbps
0000122184 68ffa58d 10.109.16.20:54679->172.217.130.70:443        122.184 Kbps
0000083968 68ffb102 10.109.16.194:6884->208.91.112.52:53          83.968 Kbps
0000044056 68ffa503 10.109.16.194:56225->208.91.112.52:53         44.056 Kbps
0000019024 68fda5df 172.26.48.39:17710->10.109.3.29:5041          19.024 Kbps
0000017008 68ffb192 10.109.19.33:45122->173.243.138.99:443        17.008 Kbps
0000014096 68c00047 172.26.48.39:7070->10.109.3.29:5008           14.096 Kbps
0000014008 68ffb695 10.109.19.96:11041->10.109.48.39:514          14.008 Kbps
0000013520 68ffa96c 10.109.16.194:56225->208.91.112.53:53         13.520 Kbps
```

Notice that the field `${rate}` is used twice on the same line: First at the very beginning of the line in plain number format as bit per second - which is used only for `sort` command, and then at the end of the line in default string format "auto-scaling" and including the units - which is intended for the user reading the output.

