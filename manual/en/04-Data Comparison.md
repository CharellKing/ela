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
  - name: compare-task
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
    action: compare
```
2. Execute the command

```bash
./ela --config ./config.yaml --tasks
```
## Detailed function explanation
### Configuration

1. Configure data comparison tasks.

| Configuration Path                    | Remarks                                                                                                              |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| /tasks/{task-name}                    | Task name, users can define multiple different tasks, and different tasks are executed sequentially.                 |
| /tasks/{task-name}/source_es          | Source ES for data comparison                                                                                        |
| /tasks/{task-name}/target_es          | Target ES for data comparison                                                                                        |
| /tasks/{task-name}/index_pairs        | Index pairs for data comparison, comparing source_index data in source ES with target_index data in target ES.       |
| /tasks/{task-name}/index_pattern      | Specifies index pairs using regular expressions; comparing index data in source ES with index data of the same name in target ES. |
| /tasks/{task-name}/slice_size         | To speed up the traversal of index data in source and target ES; data is traversed in slices; the number of slices should not be too large, generally around 20-30; set according to the capacity of the source and target ES services. slice_size determines the concurrency for querying and traversing source and target ES. |
| /tasks/{task-name}/scroll_time        | The cursor has a valid time when traversing data; if set too small, the cursor will expire; the default time is 10 minutes. |
| /tasks/{task-name}/scroll_size        | The amount of data obtained from source and target ES in one request when traversing data.                           |
| /tasks/{task-name}/buffer_count       | This parameter is the number of documents; the traversed data will be placed in a queue, and this parameter is used to set the capacity of the queue; it also determines the local memory consumption of the ela tool. For data comparison, there will be two queues, storing source ES and target ES respectively, so the final buffer usage is 2 * buffer_count. |
| /tasks/{task-name}/action_parallelism | The number of concurrent goroutines to perform data comparison.                                                      |
| /tasks/{task-name}/parallelism        | There will be many index pairs for data comparison; this parameter is used to set how many index pairs are compared concurrently within the same task. |
| /tasks/{task-name}/ids                | This parameter is not mandatory and is mainly used for data comparison of specified documents in an index; it should be set to a list of document _id fields, separated by spaces, e.g., "abc def cdf". |
| /tasks/{task-name}/action             | Set to compare, indicating that incremental data synchronization will be performed.                                  |

2. Configure Data Comparison and Repair
Often, it is desired to repair data immediately after comparison; compared to data comparison, there are the following differences:
```markdown
| Configuration Path                    | Remarks                                                                                                              |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| /tasks/{task-name}/action_size        | The unit of this parameter is M; the amount of data updated in one batch after comparison.                           |
| /tasks/{task-name}/action             | Set to sync_diff, indicating that data comparison will be performed and the differences will be repaired.            |
```
**It is worth noting that the comparison and repair function will delete data that exists in the target ES but not in the source ES.**

## Summary

When using ELA's tools, the most important thing is to ensure that it does not affect existing businesses; during testing, you should first set a relatively small value to ensure that it can work; and then continue to adjust parameters according to the service operation of ES to achieve an ideal data comparison speed.


