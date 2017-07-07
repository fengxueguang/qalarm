# qalarm
go qalarm sdk


# Install
```
 go get github.com/fengxueguang/qalarm/
```

# usage

## import
```
import "github.com/fengxueguang/qalarm/v3"
```

## Simple 
```
qalarm.NewQalarm(97, 1, 668, "err message").Send();
```
## Complicated
```
qalarm.NewQalarm(97, 1, 668, "err message", map[string]interface{}{"count":1,"serverName":"xx.com","clientIp":"127.0.0.1","script":"/opt/a/b","countType":"inc"}).Send();
```