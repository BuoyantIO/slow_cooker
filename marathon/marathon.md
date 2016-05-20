# Deploy to Marathon

```bash
dcos marathon app add marathon/slow_cooker.json
```

# View logs

```bash
open $(dcos config show core.dcos_url)/mesos
```

# Remove from Marathon

```bash
dcos marathon app remove /slow-cooker
```

# Debugging

## SSH into Mesos master

```bash
dcos node ssh --master-proxy --leader
```

## SSH into Mesos node

```bash
dcos node ssh --master-proxy --mesos-id=`dcos marathon app show /slow-cooker | jq --raw-output '.tasks[0].slaveId'`
```

## SSH into Docker container (from Mesos node)

```bash
docker exec -i -t `docker ps|grep slow_cooker|cut -f1 -d' '` bash
```
