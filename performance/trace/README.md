
#### go trace 
```shell script
$ go run trace-go.go 2 > trace.out

$ go tool trace trace.out


```

#### docker trace

```shell script
$ curl http://192.168.56.150:8080/debug/pprof/trace?seconds=20 > trace.out
$ go tool trace trace.out 
```


[og-tracer](https://tonybai.com/2021/06/28/understand-go-execution-tracer-by-example/)