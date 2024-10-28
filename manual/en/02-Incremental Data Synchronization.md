# Incremental Data Synchronization

## Quick Start
1. New Profile
```yaml
level: "info"
elastics:
    source-es5:
        addresses:
            - "http://192.168.1.55:9200"
            - "http://192.168.1.56:9200"
            - "http://192.168.1.56:9200"
        user: "es-user"
        password: "123456"
    source-es8:
        addresses:
            - "http://192.168.1.67:9200"
            - "http://192.168.1.68:9200"
            - "http://192.168.1.69:9200"
        user: "es-user2"
        password: "123456"

gateway:
    address: "0.0.0.0:8080"
    user: "user"
    password: "123456"
    source_es: "source-es5"
    target_es: "target-es8"
    master: "source-es5"
```

2. Start Ela Gateway
```bash
./ela  --config ./config.yaml --gateway
```
3. Modify the service configuration, and change the configuration of the corresponding direct ES to the corresponding address of the gateway.

```yaml
address: "<ela gateway ip>:8080"
user: "user"
password: "123456"
```

## Detailed function explanation

### Configure

1. The corresponding ES needs to be added, which refers to the source ES and the target ES here. The following is the meaning of the corresponding Key.

| Configuration Path              | Remarks                              |
| ----------------------------- | --------------------------------- |
| /elastics/{es-name}           | Add a new ES, its name is unique for easy reference in later configurations. |
| /elastics/{es-name}/addresses | You can set multiple master addresses for an ES, which can be used for load balancing. |
| /elastics/{es-name}/user      | Username of the ES                   |
| /elastics/{es-name}/password  | Password of the ES                   |

2. Add gateway configuration

| Configuration Path        | Remarks                                                                                                                               |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| /gateway/address          | Gateway listener                                                                                                                      |
| /gateway/user             | Gateway access user                                                                                                                   |
| /gateway/password         | Gateway access password                                                                                                               |
| /gateway/source_es        | Source ES for the gateway, which needs to reference the ES name in the elastics configuration; the source ES for data migration. This not only means the data source of the migration, but also determines that ELA will parse the API according to the corresponding version of ES and return the data format of the corresponding version of ES. |
| /gateway/target_es        | Target ES for the gateway, which needs to reference the ES name in the elastics configuration; the target ES for data migration.       |
| /gateway/master           | Master ES, which can only be selected from /gateway/source_es and /gateway/target_es. It determines the ES that currently prioritizes synchronous read and write. The other ES will be the slave, which will only have write requests, and the business side will not perceive the success or failure of the slave requests. |

### Master-slave switching
At the beginning,/gateway/master should be set to the ES referenced by/gateway/source_es; because the inventory data has not been migrated yet, only the source ES has full data; after the inventory data migration is completed,/gateweay/master can be migrated to/gateway/target_es, that is, the target ES.

Switching ES is simple, just set/gateway/master to the ES referenced by/gateway/target_es.

```yaml
gateway:
    ...
    source_es: "source-es5"
    target_es: "target-es8"
    master: "target-es8"
```

### Summary

This is the advantage of Ela Gateway, which helps to make API differences between different versions of Elasticsearch compatible, and the intrusion to business code is basically zero; it can smoothly switch between old and new Elasticsearch without stopping the service.

