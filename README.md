# harbor-replicator

Replicate local Harbor(https://github.com/vmware/harbor) docker images to public cloud. 

## CMD
```
$ go build
$ ./harbor-replicator --help
Usage of ./harbor-replicator:
  -harbor string
        harbor registry server address
  -insecure
        using http:// scheme for harbor
  -pass string
        password for harbor
  -project string
        filter projects
  -remote string
        remote registry
  -remote_pass string
        password for remote registry
  -remote_user string
        user for remote registry
  -user string
        user for harbor
  ```
  
# Environment
```bash
export verbose=1    # verbose docker command output
export timeout=300s # docker command timeout setting, default 180s

```
