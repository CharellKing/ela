# Stock Data Migration

## Quick Start

1. Create a new configure

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

tasks:
  - name: stock-sync-task
    source_es: es5
    target_es: es8
    index_pairs:
      -
        source_index: "hello_sample"
        target_index: "sample_123"
    index_pattern: "test_.*"
    force: true
    slice_size: 5
    scroll_size: 5000
    buffer_count: 3000
    action_parallelism: 5
    action_size: 5
    scroll_time: 10
    parallelism: 12
    action: sync
```
2. Execute the command to perform data synchronization

```bash
./ela --config ./config.yaml --tasks
```

## Detailed function explanation
### Configuration

Configure the stock data migration tasks

| Configuration Path                    | Remarks                                                                                                              |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| /tasks/{task-name}                    | Task name, users can define multiple different tasks, and different tasks are executed sequentially.                 |
| /tasks/{task-name}/source_es          | Source ES for data migration                                                                                         |
| /tasks/{task-name}/target_es          | Target ES for data migration                                                                                         |
| /tasks/{task-name}/index_pairs        | Index pairs for data migration, migrating source_index from source ES to target_index in target ES.                  |
| /tasks/{task-name}/index_pattern      | Specifies index pairs using regular expressions; data from indexes in source ES will be migrated to indexes with the same name in target ES. |
| /tasks/{task-name}/force              | Whether to forcibly delete indexes in target ES before full data synchronization and copy index configurations from source ES to target ES. If set to False, indexes with the same ID will be updated in target ES, overwriting the index data in target ES. |
| /tasks/{task-name}/slice_size         | To speed up the traversal of index data in source ES; data is traversed in slices; the number of slices should not be too large, generally around 20-30; set according to the capacity of the source ES service. slice_size determines the concurrency for querying and traversing source ES. |
| /tasks/{task-name}/scroll_time        | The cursor has a valid time when traversing data; if set too small, the cursor will expire; the default time is 10 minutes. |
| /tasks/{task-name}/scroll_size        | The amount of data obtained from source ES in one request when traversing data.                                       |
| /tasks/{task-name}/buffer_count       | The traversed data will be placed in a queue, and this parameter is used to set the capacity of the queue; it also determines the local memory consumption of the ela tool. |
| /tasks/{task-name}/action_parallelism | The traversed data will be written to target ES in batches; this parameter is used to determine the concurrency for writing. |
| /tasks/{task-name}/action_size        | The unit of this parameter is M; when the body size of the batch write interface reaches this value, a batch write operation will be performed. |
| /tasks/{task-name}/parallelism        | There will be many index pairs for data synchronization; this parameter is used to set how many index pairs are synchronized concurrently within the same task. |
| /tasks/{task-name}/ids                | This parameter is not mandatory and is mainly used for data synchronization of specified documents in an index; it should be set to a list of document _id fields, separated by spaces, e.g., "abc def cdf". |
| /tasks/{task-name}/action             | Set to sync, indicating that full data synchronization will be performed.                                             |


## Summary

When using ELA's tools, the most important thing is to ensure that it does not affect existing businesses; during testing, you should first set a relatively small value to ensure that it can work; and then continue to work according to the service operation of ES. Adjust parameters to achieve an ideal data migration speed.

