# ela

Ela is a tool to migrator data between different elasticsearch with different versions. Elasticsearch above 5.x.x version is supported.

## Usage

```bash
./ela ./config.yml
```

## Config

```yaml
level: "info"         # log level which is used to log message, can be assigned to 'debug', 'info', 'warn', 'error'.
elastics:             # elasticsearch clusters
  es5:                # cluster name which is unique
    addresses:        # elasticsearch addresses which has many masters
      - "http://127.0.0.1:15200"
    user: ""          # basic auth username
    password: ""      # basic auth password

  es6:
    addresses:
      - "http://127.0.0.1:16200"
    user: ""
    password: ""

  es7:
    addresses:
      - "http://127.0.0.1:17200"
    user: ""
    password: ""

  es8:
    addresses:
      - "http://127.0.0.1:18200"
    user: ""
    password: ""

tasks:  # tasks which is executed orderly.
  - name: task1 # task name
    source_es: es5  # source elasticsearch cluster name which is defined in elastics config
    target_es: es8  # target elasticsearch cluster name which is defined in elastics config 
    index_pairs:    # index multiple pairs which is used to sync data from source index to target_index
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    index_pattern: "test_.*" # index pattern which is used to filter index to sync, source index is same with target index.
    action: sync # index actions which can be assigned to 'sync', 'compare', 'sync_diff'. sync to insert data, compare to compare source index with target index, sync_diff to sync data between source index and target index.
    force: true       # force to cover the target index data with source index data and settings.
    scroll_size: 1000 # scroll size which is used to scroll data from source index.
    scroll_time: 10   # scroll time which is used to scroll data from source index, default is 10 minutes.
    parallelism: 12   # parallelism which is used to sync data in parallel index pairs.

  - name: task2
    source_es: es5
    target_es: es8
    index_pairs:
      - source_index: "sample_hello"
        target_index: "sample_hello"
    action: sync
    force: true

  - name: task3
    source_es: es5
    target_es: es6
    index_pairs:
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    action: compare
    force: true

  - name: task3
    source_es: es5
    target_es: es6
    index_pairs:
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    action: sync_diff
    force: true
```

