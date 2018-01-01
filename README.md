### Purpose
To experiment with a Golang-based parser generator for translating string-based search queries into JSON suitable for use with Elasticsearch.
Currently using `github.com/pointlander/peg` for parsing and `gopkg.in/olivere/elastic.v5` for ES5-compatible query rendering.
The goal here is to play with the parser generator, not generate optimally-structured ES queries. The DSL can generate valid ES queries
that are not particularly performant or readable. Search results returned for a deeply nested query can be counterintuitive.

#### TODOs
Life is short and this tool has no practical use, but for fun it would be nice to also:
* upgrade AST traversal business logic and grammar definitions to flatten/simplify generated queries
* move value element parsing from the grammar into the traversal logic to provide more actionable error messages
* support list-type values in queries, more query types etc.
* add method for setting or defaulting various query params that have no clear place in such a DSL


### Build & Run
* Build the binary using `make` or `make clean build`
*  Binary will be compiled into `dist/` dir, with generated parser generated to file `grammar/dsl.peg.go`
* Try `dist/es_dsl --help` after running `make`, for detailed usage instructions
* Example invocation: `dist/es_dsl --verbose --filter --default 'message' --query 'DSL_QUERY_STRING'`


### DSL Grammar
Instructions for using the DSL are [here](https://github.com/elireisman/go_es_query_parser/blob/master/grammar/README.md)


### Tips & Gotchas
* The `--verbose` flag will display the full parse tree before rendering the final ES query JSON
* `AND` operator binds tighter than `OR` in the absense of groupings (as expected)
* A query (base or nested) defaults to `AND` behavior until the leftmost `OR` is encountered
* Try piping the tool's output through `| tail -1 | jq .` for pretty-printed output

