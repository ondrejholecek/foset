# Foset internal plugin: stats

Plugin to generate many top-like graphs from various areas, like:
- Number of sessions per VDOMs, policies, ...
- Sessions by IP protocols
- Authenticated sessions by users
- Communication from/to/between ports and networks
- Number of sessions in different TCP states
- Packets and bytes exchanged between sessions
- Data rates from/to/between networks
- Sessions coming from or going to different interfaces
- Session fully or partially offloaded
- ... and more

The statistics in the output are quite general by design It is expected that the user manually runs Foset with the filter 
(`-f`) that is required. It can be even executed several times with different settings and outputs aggregated in the
same output directory (see bellow).

## Screenshots

### Starting tab - General overview
![Example HTML - General overview](/plugins/stats/README.img/general.png)

### TCP and UDP ports tab
![Example HTML - TCP/UDP ports](/plugins/stats/README.img/ports.png)

### Session coming from and going to different interfaces
![Example HTML - Interfaces](/plugins/stats/README.img/interfaces.png)

## Output 

The output is a completely independent HTML document (actually a directory) that can be copied and used even without
any Internet access.

The name of the output directory is specified with `directory` parameter. If such directory does not exist, it is created
together with all the necessary files inside. If it exists, only the new graph data are updated.

**Be careful when using already existing directories that do not contain the all the necessary files, because the plugin does 
not create them if the directory already exists (if `force` parameter was not used) and in that case it is not possible to 
display the results (or at lease not properly).**

## HTML GUI

The (automatically created) directory contains `index.html` which is the main HTML file that should be opened 
in the web browser to see the GUI of the application. There is also a sub-directory `resources` which contains all the files
and scripts necessary to show the output.

To navigate in the GUI mouse(or touchscreen) can be used. To make it easier to compare graphs from different executions,
there are also many keyboard shortcuts configured. Keyboard can be used to switch between execution or between tabs.
It is also possible to "bookmark" specific executions and quickly switch between them without having to go through the
other execution outputs. To learn the shortcuts, see the "Help" button in the top right corner.

## Group many executions results into one output directory

One directory can hold and display unlimited amount of foset results. This is usually used to group outputs from one session 
dump file executed with different filters, or two related session dumps (like HA master/slave or dumps from the same
FortiGate but collected in different times).

All graphs are always calculated from one execution only, but at top of the page there is a select-box to switch between
different execution results.

## Mapping indexes to real names

The plugin can use the real VDOM and interface names it is it initialized after [indexmap plugin](/plugins/indexmap/indexmap.md)
**and** the parameters `transvdoms` and/or `transifaces` were specified.

## IP based statistics

For the statistics related to IP addresses, by default the hosts are grouped by /24 networks they belong to. Different
prefix length can be specified with `srcprefix` and/or `dstprefix` parameters, however be aware that by increasing the
statistics prefix length, the memory consumption increases as well.

## Complex statistics

The IP (networks) and TCP/UDP ports statistics are usually shown in two graphs - one for source networks or ports and
one for destination networks or ports. It is also possible to generate additional graphs showing the communication
between source and destination networks/ports but this is not always meaningful and uses extreme amount of RAM, therefore
those graphs are disabled by default. To enable generating them, use the parameter `complex`.

## Example

If you are starting with a new statistics, delete the whole output directory, so the plugin can create and populate it
correctly from scratch:

```
rm -rf /tmp/example
```

Then run Foset with stats plugin as many times as you want with different filter to have all the executions in the same
output directory:

```
$ ./foset -r ~/tmp/core -p 'stats|directory=/tmp/example'
$ ./foset -r ~/tmp/core -p 'stats|directory=/tmp/example' -f 'host in 8.8.0.0/16'
$ ./foset -r ~/tmp/core -p 'stats|directory=/tmp/example' -f 'dport < 1024'
$ ./foset -r ~/tmp/core -p 'stats|directory=/tmp/example' -f 'dport < 1024 and not offloaded'
```

Open the `/tmo/example/index.html` file in your favourite browser.

![Example HTML - Executions selection](/plugins/stats/README.img/filters.png)

