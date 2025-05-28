<h4 align="right"><strong>English</strong> | <a href="https://github.com/CharellKing/ela/blob/master/README_CN.md">简体中文</a></h4>
<p align="center">
    <img src=./logo.png width=138/>
</p>
<h1 align="center">Ela</h1>
<p align="center"><strong>A tool to migrate data between elasticsearch <em>without loss</em>.</strong></p>


## Features
1. copy index settings between elasticsearch.
2. batch create index template according elasticsearch index.
3. sync stock data from source elasticsearch.
4. compare data between elasticsearch.
5. compare & sync data between elasticsearch.
6. import data from file to elasticsearch.
7. export data from elasticsearch to file.
8. sync incremental data without service loss.

## Compile

```bash
go mod tidy
go build -o ela
```

## Usage

### Configuration

```yaml
level: "info"
elastics:
    es5:
        addresses:
            - "http://192.168.1.55:9200"
            - "http://192.168.1.56:9200"
            - "http://192.168.1.56:9200"
        user: "es-user"
        password: "123456"
    es8:
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
gateway:
    address: "0.0.0.0:8080"
    user: "user"
    password: "123456"
    source_es: "source-es5"
    target_es: "target-es8"
    master: "source-es5"
```

### Start Ela Gateway

```bash
./ela --config ./config.yaml --gateway
```

### Start the stock data migrate task

```bash
./ela --config ./config.yaml --task "stock-sync-task"
```

## Manual
Please refer to [Manual](https://ela-doc.memehub.info/docs/) for more details.



