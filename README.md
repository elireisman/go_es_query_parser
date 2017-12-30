### Purpose
To experiment with a Golang-based parser generator for translating string-based search queries into JSON suitable for use with Elasticsearch.
Currently using `github.com/pointlander/peg` for parsing and `gopkg.in/olivere/elastic.v5` for ES5-compatible query rendering.
This approach allows for parsing arbitrarily complex queries, and to generate reasonable parse and type errors for DSL users.
_This is WIP, is subject to (rapid) change, and is not expected to work as advertised at any given moment_.

### Build & Run
1. Build the binary using `make` or `make clean build`
2. Binary will be compiled into `dist/` dir, with generated parser generated to file `grammar/dsl.peg.go`
3. Currently generates a generic `bool` query, or using `--filter` arg, can translate your DSL query into the filtered clause of a `match_all` `bool` query
4. `--verbose` argument will dump additional state info during & after parsing, to aid DSL query debugging
5. `--help` will print the usage string (TBA a detailed breakdown of the grammar rules)

Example invokations:
1. `dist/dsl_to_es_json_parser --query 'YOUR_QUERY_STRING'`
2. `dist/dsl_to_es_json_parser --query 'YOUR_QUERY_STRING' --filter`
3. `dist/dsl_to_es_json_parser --help`


### DSL Grammar
_TODO: add grammar breakdown to CLI --help usage dump, or detail it here_

Example DSL queries:
* Simple token or double-quoted phrase treated as match or match-phrase subquery
`foo AND bar`
`"lorem ipsum goo goo gajoob" OR (bar AND baz)`
`NOT a OR (b AND (c OR d OR (e AND NOT f))) OR (NOT (x AND y) OR z)`

* Colon-separated key value pairs map to term, match\_phrase, range, or exists query on a particular document field and value
`foo:bar AND baz:123 AND x:>=50 AND msg:"some phrase blah blah" AND some_field:?`
`(x AND y) OR z:>=55 OR post_fools_day:>2017/04/01`
`abc AND "one two three" AND g:"gee whiz" AND ((baz:<2017/01/01 AND bar:>=2017/10/31) OR foo:<=7)`


### Learnings So Far
* `pointlander/peg` generator produces clean, readable code you can investigate directly to debug errors in your grammar or a particular parse run
* `pointlander/peg`'s error messages are ambiguous; warnings are often not localized to the site of the error in the grammar (project has open issues for this)
* There is a bug when importing `gopkg.in` based stable releases of vendored dependencies directly into a `*.peg` grammar file that required workarounds
* Mixing `olivere/elastic` releases for multi-version support will require a script to manually vendor and shade (I used `sed -i ...`) various lib versions internally
