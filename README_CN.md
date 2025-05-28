<h4 align="right"><strong>简体中文</strong> | <a href="https://github.com/CharellKing/ela/blob/master/README.md">English</a></h4>
<p align="center">
    <img src=./logo.png width=138/>
</p>
<h1 align="center">Ela</h1>
<p align="center"><strong>一套<em>不用停服</em>的数据迁移方案.</strong></p>


## Features
1. 拷贝索引配置.
2. 批量根据索引配置创建索引模板.
3. 同步存量数据.
4. 仅比对数据.
5. 同步并且比对数据
6. 从文件导入数据到索引.
7. 将索引数据导出到文件.
8. 无损的增量数据同步.

## 编译

```bash
go mod tidy
go build -o ela
```

## 使用

### 配置文件

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


### 启动 Ela Gateway

```bash
./ela --config ./config.yaml --gateway
```

### 启动数据存量同步

```bash
./ela --config ./config.yaml --task "stock-sync-task"
```

## 手册

详情请查看 [Ela 手册](https://ela-doc.memehub.info/zh-cn/docs/)


