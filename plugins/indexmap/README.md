# Foset internal plugin: indexmap

This internal plugin is used to map certain fields in the Session entry that are only represented by the index number
there to a human readable real name.

For that the outputs of some additional commands collected on the same FortiGate are needed.

At this moment following mapping of VDOM ids to VDOM names and mapping of interface ids to names are supported. Either 
one of them can be used or both at the same time. If both `vdom` and `interfaces` parameters are used, they can point
to the same file, however neither of them should point to the same file as the output of `diagnose sys session list`,
because for big session dumps, it can significantly slow down the plugin initialization.

## VDOM mapping

Parameter `vdom` must be provided with the value as path to the file containg output of the command `diagnose sys vd list`. 
The command itself does not need to be in the text file, because the right section is recognized by the line
`list virtual firewall info:`.

VDOM name is saved in the custom variable `vdom` which is completely independent on the regular Session variable `vdom`
contaiting the original VDOM id. If the VDOM name is not found, the original VDOM index is stored as a string.

Sample file can look like this:

```
fgt # diagnose sys vd list
system fib version=396
list virtual firewall info:
name=root/root index=0 enabled fib_ver=365 use=4896 rt_num=4634 asym_rt=0 sip_helper=1, sip_nat_trace=1, mc_fwd=0, mc_ttl_nc=0, tpmc_sk_pl=0
ecmp=source-ip-based, ecmp6=source-ip-based asym_rt6=0 rt6_num=153 strict_src_check=1 dns_log=1 ses_num=4996 ses6_num=0 pkt_num=3262773641
	tree_flag=1 tree6_flag=1 nataf=0 traffic_log=1 extended_traffic_log=0 svc_depth=38
	log_neigh=0, deny_tcp_with_icmp=0 ses_denied_traffic=no tcp_no_syn_check=0 central_nat=0 policy_mode_ngfw=0 block_land_attack=0 link_check_local_in=1
	fw_session_hairpin=no  keep-PRP-trailer=0
	ipv4_rate=24, ipv6_rate=0, mcast6-PMTU=0, allow_linkdown_path=0
	per_policy_disclaimer=0
	mode=AP ha_state=work prio=0 vid=0
name=vsys_ha/vsys_ha index=1 enabled fib_ver=13 use=54 rt_num=6 asym_rt=0 sip_helper=0, sip_nat_trace=1, mc_fwd=0, mc_ttl_nc=0, tpmc_sk_pl=0
ecmp=source-ip-based, ecmp6=source-ip-based asym_rt6=0 rt6_num=8 strict_src_check=0 dns_log=0 ses_num=13 ses6_num=0 pkt_num=225720795
	tree_flag=1 tree6_flag=0 nataf=0 traffic_log=0 extended_traffic_log=0 svc_depth=1
	log_neigh=0, deny_tcp_with_icmp=0 ses_denied_traffic=no tcp_no_syn_check=0 central_nat=0 policy_mode_ngfw=0 block_land_attack=0 link_check_local_in=1
	fw_session_hairpin=no  keep-PRP-trailer=0
	ipv4_rate=0, ipv6_rate=0, mcast6-PMTU=0, allow_linkdown_path=0
	per_policy_disclaimer=0
	ha_flags={no-ses-sync,no-ses-flush,no-ha-stats} mode=standalone ha_state=work prio=0 vid=0
name=vsys_fgfm/vsys_fgfm index=2 enabled fib_ver=4 use=41 rt_num=0 asym_rt=0 sip_helper=0, sip_nat_trace=1, mc_fwd=0, mc_ttl_nc=0, tpmc_sk_pl=0
ecmp=source-ip-based, ecmp6=source-ip-based asym_rt6=0 rt6_num=4 strict_src_check=0 dns_log=0 ses_num=0 ses6_num=0 pkt_num=0
	tree_flag=0 tree6_flag=0 nataf=0 traffic_log=0 extended_traffic_log=0 svc_depth=0
	log_neigh=0, deny_tcp_with_icmp=0 ses_denied_traffic=no tcp_no_syn_check=0 central_nat=0 policy_mode_ngfw=0 block_land_attack=0 link_check_local_in=1
	fw_session_hairpin=no  keep-PRP-trailer=0
	ipv4_rate=0, ipv6_rate=0, mcast6-PMTU=0, allow_linkdown_path=0
	per_policy_disclaimer=0
	mode=standalone ha_state=work prio=0 vid=0
vf_count=4 vfe_count=0
```

### Example

```
$ foset -r /tmp/sessions.txt -p 'indexmap|vdoms=/tmp/vdoms.txt' \
  -o '${sdap:-40s} : ${vdom} (${custom|vdom})'

193.86.26.197:1884->8.8.4.4:53           : 0 (root)
10.109.3.14:37327->205.251.194.229:53    : 0 (root)
10.109.248.18:49991->10.109.3.14:53      : 0 (root)
10.109.3.14:40015->205.251.197.78:53     : 0 (root)
10.109.3.18:57921->10.109.3.254:161      : 0 (root)
172.26.48.39:17711->10.109.3.29:5041     : 0 (root)
10.109.16.133:45113->8.8.8.8:53          : 0 (root)
```


## Interface mapping

To map the interface index to its name, the output of the command `diagnose netlink interface list` must be in the file
name specified in the parameter `interfaces`. Again the command itself does not need to be inside the file, because
the interface definition is recognized by the line starting with `if=` and having the column `index=` later on the same
line.

Interface name(s) are saved in the custom fields `iface[oi]`, `iface[oi]`, `iface[oo]`, `iface[ri]`, `iface[ir]`, 
`iface[ro]`, `iface[or]` which have the same meaning as the similar fields in the
[formatter string](https://github.com/ondrejholecek/fortisession/blob/master/fortiformatter/output_format.md), but are 
completely independent on them. If the interface name is not found, the original index number is stored as a string.

File's contents should be like following one:

```
[...]
if=port1 family=00 type=1 index=9 mtu=1500 link=0 master=0
ref=24 state=off start fw_flags=0 flags=up broadcast run promsic multicast

if=port2 family=00 type=1 index=10 mtu=1500 link=0 master=0
ref=21 state=off start present fw_flags=0 flags=up broadcast promsic multicast
[...]
if=Internet family=00 type=1 index=54 mtu=1500 link=0 master=0
ref=2192 state=off start fw_flags=0 flags=up broadcast run promsic master multicast

if=Management family=00 type=1 index=55 mtu=1500 link=0 master=0
ref=495 state=off start fw_flags=0 flags=up broadcast run promsic multicast

[...]
```

### Example

```
$ foset -r /tmp/sessions.txt -p 'indexmap|interfaces=/tmp/interfaces.txt' \
  -o '${sdap:-40s} : ${iface[oo]} (${custom|iface[oo]})'
  
193.86.26.197:1884->8.8.4.4:53           : 54 (Internet)
10.109.248.18:49991->10.109.3.14:53      : 55 (Management)
10.109.3.18:57921->10.109.3.254:161      : 52 (root)
172.26.48.39:17711->10.109.3.29:5041     : 55 (Management)
10.109.250.108:52120->45.75.200.85:53    : 54 (Internet)
```

## Using both VDOMs and interfaces mapping at the same time

```
$ foset -r /tmp/sessions.txt -p 'indexmap|vdoms=/tmp/vdoms.txt,interfaces=/tmp/interfaces.txt' \
  -o '${sdap:-40s} : ${vdom} (${custom|vdom}) ${iface[oo]} (${custom|iface[oo]})'

172.253.14.1:44137->193.86.26.196:53     : 0 (root) 55 (Management)
172.253.2.1:43145->193.86.26.196:53      : 0 (root) 55 (Management)
10.109.248.18:123->5.1.56.123:123        : 0 (root) 54 (Internet)
10.109.19.162:56983->10.109.3.14:53      : 0 (root) 55 (Management)
10.109.250.110:32173->208.91.113.70:123  : 0 (root) 54 (Internet)
193.86.26.197:1884->8.8.4.4:53           : 0 (root) 54 (Internet)
10.109.248.18:49991->10.109.3.14:53      : 0 (root) 55 (Management)
```
