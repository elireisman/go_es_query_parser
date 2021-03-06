package main


import (
  "encoding/json"
  "fmt"
  "flag"
  "log"
  "os"

  "gopkg.in/olivere/elastic.v5"

  "github.com/elireisman/go_es_query_parser/grammar"
  "github.com/elireisman/go_es_query_parser/utils"
)

const NoInput = "ERR_NO_INPUT_PROVIDED"

func main() {
  // reqister and parse CLI args
  query := flag.String("query", NoInput, "the query (written in the DSL) you wish to submit")
  isFilter := flag.Bool("filter", false, "structure the output as a filtered match_all instead of standard query")
  verbose := flag.Bool("verbose", false, "log/explain verbosely during parsing")
  defField := flag.String("default", "_all", "select a default field for non-KV values to applied against in the final query")
  defOper := flag.Bool("default-or", false, "override default query clause operator AND, use OR instead")
  halp := flag.Bool("help", false, "print DSL and usage details and exit")
  flag.Parse()

  if *halp {
    log.Println(usage())
    os.Exit(1)
  }
  if *query == NoInput {
    log.Println("-query argument specifying query string is required, aborting")
    os.Exit(1)
  }

  // init DSL state object and parse the input
  dsl := &grammar.DSL2ES{
    Queries:    &utils.QueryStack{},
    Values:     &utils.ValueStack{},
    Verbose:    *verbose,
    IsFilter:   *isFilter,
    Buffer:     *query,
  }

  dsl.Init()
  dsl.Queries.Init(*defOper)
  dsl.Values.Init(*defField)
  if err := dsl.Parse(); err != nil {
    log.Fatalf("[ERROR] Parsing input, err=%s", err)
  }

  // if --verbose flag, let's see the AST before proceeding to translation step
  if *verbose {
    dsl.PrintSyntaxTree()
  }

  // walk the parsed AST, firing off the logic in the rule/token actions, resulting in ES query/filter tree
  dsl.Execute()
  if dsl.Queries.Output == nil {
    log.Fatalf("parsing of query %q failed, no output registered, aborting", *query)
  }

  // render final query/filter output
  var rendered interface{}
  var err error
  if dsl.IsFilter {
    rendered, err = elastic.NewBoolQuery().Filter(dsl.Queries.Output).Source()
  } else {
    rendered, err = dsl.Queries.Output.Source()
  }
  if err != nil {
    log.Fatalf("query rendering for %q failed, err=%s", *query, err)
  }

  // display results as JSON string, ready to use
  j, err := json.Marshal(rendered)
  if err != nil {
    log.Fatalf("failed to marshal rendered query as JSON, err=%s", err)
  }
  fmt.Println(`{"query":` + string(j) + `}`)
}

func usage() string {
  return fmt.Sprintf("Usage: %s --query 'QUERY_STRING' [--filter] [--verbose] [--help]", os.Args[0])
  // TODO: detail the DSL grammar etc. here also, or with verbose + help opts together only?
}
