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

你应该按照如下的文章顺序阅读，方便了解整套方案是如何运作的。

1. [数据迁移总体方案](manual%2Fcn%2F01-Elasticsearch%20%E6%95%B0%E6%8D%AE%E8%BF%81%E7%A7%BB%E6%80%BB%E4%BD%93%E6%96%B9%E6%A1%88.md)
2. [增量数据同步](manual%2Fcn%2F02-%E5%A2%9E%E9%87%8F%E6%95%B0%E6%8D%AE%E5%90%8C%E6%AD%A5.md)
3. [存量数据迁移](manual%2Fcn%2F03-%E5%AD%98%E9%87%8F%E6%95%B0%E6%8D%AE%E8%BF%81%E7%A7%BB.md)
4. [数据比对](manual%2Fcn%2F04-%E6%95%B0%E6%8D%AE%E6%AF%94%E5%AF%B9.md)


