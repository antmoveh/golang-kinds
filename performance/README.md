


#### go pprof

```shell script
$  go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap
```

#### Linux FlameGraph

```shell script
# cpu
$ git clone https://github.com/brendangregg/FlameGraph
$ cd FlameGraph
$ perf record -F 99 -g -p 17054 -- sleep 30
$ perf script | ./stackcollapse-perf.pl > out.perf-folded
$ ./flamegraph.pl out.perf-folded > perf.svg
$ firefox perf.svg 
```

```shell script
# meory
$ git clone https://ghproxy.com/https://github.com/brendangregg/BPF-tools.git
$ cd BPF-tools/old/2017-12-23
$ ./mallocstacks.py -f 30 > out.stack

$ cd FlameGraph
$ ./flamegraph.pl --color=mem --title="malloc() bytes Flame Graph" --countname=bytes < out.stacks > out.svg

```