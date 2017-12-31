### The Query DSL

The DSL is comprised of operators (`AND`, `OR`, and `NOT`), grouping symbols (parentheses), and value elements.
Values are comprised of either a single value (applied to the search query against a default field.) or a key-value
pair, separated by a `:` (colon.) In the latter case, the keys are a single token representing a field in
the indexed documents you wish to search. Values are comprised of one of a number of possible data types,
corresponding to various query types.

Some value element examples:

`foo` ~ search the default field for term "foo"

`35` ~ search the default field for the number 35, as an integer in a term query

`name:Joe` ~ search the `name` field for the value "Joe" as a term query

`count:2` ~ search the `count` field for the numerical value 2 as a term query

`graduated:?` ~ search for documents where the `graduated` field exists

`msg:"foo bar baz"` ~ search the `msg` field using a match-phrase query

`amount:>=40` ~ search the `amount` field using a range query for documents where the field's value is greater than or equal to 40

`created_at:<2017-10-31` ~ search the `created_at` field for dates before Halloween, 2017


Any field or parenthesized grouping can be negated with the `NOT` or `!` operator:

`NOT foo` ~ search for documents where default field doesn't contain the token `foo`

`NOT available:?` ~ search for documents where `available` field does not exist 

`NOT count:>100` ~ search for documents where `count` field has a value that's _not_ greater than 100

`NOT (x OR y)` ~ search the default field for documents that don't contain terms "x" or "y"


Parentheses are used for grouping of subqueries:

`a OR (b:"some words" AND NOT c:20)` ~ return docs containing term "a" or where field `b` matches the phrase "some words", but field `c`'s value is not 20.

`NOT foo:bar AND baz:99` ~ return docs where field `foo`'s value is not "bar" and where field `baz`'s value is 99.

Nesting depth is arbitrary, limits are configured on the ES side:

`(a OR b OR (c:5 AND d:10)) AND NOT ((x:foo OR x:bar) AND y:?) AND NOT z:?`

