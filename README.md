# ela

Ela is a tool to migrator data between different elasticsearch with different versions. Elasticsearch above 5.x.x version is supported.

## Features
1. copy index settings and mappings from source index to target index.
2. create index template from source index to target index.
3. full sync data from source es to target es.
4. incremental sync data from source es to target es.
5. compare data between source es and target es.
6. import data from index file to target es index.
7. export index data from source es to index file.

## Usage

+ run tasks
```bash
./ela --config ./config.yml --tasks
```

+ run gateway
```bash
./ela --config ./config.yml --gateway
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
  - name: full-sync-task # task name
    source_es: es5  # source elasticsearch cluster name which is defined in elastics config
    target_es: es8  # target elasticsearch cluster name which is defined in elastics config 
    index_pairs:    # index multiple pairs which is used to sync data from source index to target_index
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    index_pattern: "test_.*" # index pattern which is used to filter index to sync, source index is same with target index.
    action: sync # index actions which can be assigned to 'sync', 'compare', 'sync_diff'. sync to insert data, compare to compare source index with target index, sync_diff to sync data between source index and target index.
    force: true       # force to cover the target index data with source index data and settings.
    slice_size: 5 # search with slice number which is the parallelism size, default is 20.
    scroll_size: 5000 # scroll size which is used to scroll data from source index, default is 10000.
    buffer_count: 3000 # buffer count which is used to buffer data to sync, default is 100000.
    action_parallelism: 5 # write parallelism which is used to write data, default is 10.
    action_size: 5 # bulk data when reach the size, default is 5MB, default is 5.
    scroll_time: 10   # scroll time which is used to scroll data from source index, default is 10 minutes.
    parallelism: 12   # parallelism which is used to sync data in parallel index pairs, default is 12.
    

  - name: increment-sync-task
    source_es: es5
    target_es: es8
    index_pairs:
      - source_index: "sample_hello"
        target_index: "sample_hello"
    action: sync_diff

  - name: compare-task
    source_es: es5
    target_es: es6
    index_pairs:
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    action: compare

  - name: copy-index
    source_es: es5
    target_es: es6
    index_pairs:
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    action: copy_index

  - name: copy-index-task
    source_es: es5
    target_es: es6
    index_pairs:
      -
        source_index: "sample_hello"
        target_index: "sample_hello"
    action: copy_index

  - name: import-task
    target_es: es6
    index_pattern: "test_*" # import test prefix index data from index data file.
    index_file_root: "C:/Users/andy/Documents" # index file root which is used to import index data.
    index_pairs:
      -
        index: "sample_hello" # import data from index file dir to sample_hello index.
        index_file_dir: "C:/Users/andy/Documents/abc"
    action: import

  - name: export-task
    source_es: es5
    index_pattern: "test_*" # export test prefix index data to file.
    index_file_root: "C:/Users/andy/Documents" # index file root which is used to export index data.
    index_pairs:
      -
        index: "sample_hello"  # export sample_hello index data to index file dir.
        index_file_dir: "C:/Users/andy/Documents/abc"
    action: export

  - name: create-template
    source_es: es5
    target_es: es6
    action: create_template # create index template from source index to target index.
    index_templates:
      -
        name: "template_sample_hello"
        pattern: ["sample_*", "index_*"]  # index pattern which is used to filter index to create template.
        order: 0

gateway: # gateway for dual write both source es and target es.
  gateway_address: "0.0.0.0:8080"  # gateway listen address
  gateway_user: "user"             # gateway auth username
  gateway_password: "12342"        # gateway auth password
  source_es: es5                   # gateway source es
  target_es: es8                   # gateway target es
  master: es8                      # gateway master es, select between source es and target es.

```


