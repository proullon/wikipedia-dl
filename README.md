## Wikipedia to CockroachDB

This is an utilitary tool to import any Wikipedia language into your CockroachDB cluster.

To avoid hammering your cluster, wikipediatocrdb used [workerpool](https://github.com/proullon/workerpool) to adapt parallelisation in light of insert speed.

# .dev.conf

To use Makefile rules, populate .dev.conf file (from .dev.conf.example)

## Binary requirements

* wget
* bzip2

## Parameters

* language: set language (default en)
* interactive: select which dumps will be imported
* dump-folder: download and extraction folder
* tight: remove dump after import
* with-page-content: insert wikipedia article body
* with-page-reference: populate `article_references` table

## Documentation

* https://en.wikipedia.org/wiki/Wikipedia:Database_download
* https://dumps.wikimedia.org/enwiki/latest/
* https://www.cockroachlabs.com/blog/serializable-lockless-distributed-isolation-cockroachdb/
