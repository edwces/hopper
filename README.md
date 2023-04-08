# Web crawler in Go

## Requirements:
- given a set of urls visits them and store page data
- pages are stored in some kind of storage
- urls in these pages are extracted
- now they are appended to frontier and the process repeats

## Analasis:
- system needs to be optimized for speed for billions of site
that can be run. We should use Parraller computing
- should handle edge cases with badly formatted html
- it should be easily configurable while being simple and easy to use
- extensible for other media formats
- respect owner sites(policies e.g. politeness)
- needs tracing for ease of use
- needs to make frequent backups if data gets corrupted

## Parts
1. Frontier - data structure responsible for storing currently processed urls
2. Fetcher - responsible for downloading pages
3. Parser - HTML Parser
4. Storage - some storage(e.g. Sqlite) for storing pages
5. URL extractor - extracts and normalizes urls from parsed HTML tree
6. URL Filterer - filters out malicious urls or urls that should not have been processed(from robots.txt)
7. URL Deduplicator - filters out urls which already have been seen by crawler
8. URL Loader - loads most prioritized urls into Frontier

* Also caches for Parser, Fetcher
