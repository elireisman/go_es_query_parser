package main


import (
  "encoding/json"
  "fmt"
  "flag"
  "log"
  "os"

  "gopkg.in/olivere/elastic"

  "github.com/elireisman/go_es_query_parser/grammar"
  "github.com/elireisman/go_es_query_parser/utils"
)

const NoInput = "ERR_NO_INPUT_PROVIDED"

func main() {
  // reqister and parse CLI args
  query := flag.String("query", NoInput, "the query (written in the DSL) you wish to submit")
  isFilter := flag.Bool("filter", false, "structure the output as a filtered match_all instead of standard query")
  verbose := flag.Bool("verbose", false, "log/explain verbosely during parsing")
  halp := flag.Bool("help", false, "print DSL and usage details and exit")
  flag.Parse()

  if *query == NoInput {
    log.Println("-query argument specifying query string is required, aborting")
    os.Exit(1)
  }
  if *halp {
    log.Println(usage())
    os.Exit(1)
  }
  log.Printf("Query: %q\t\t(is filter= %t, verbose=%t, help=%t)", *query, *isFilter, *verbose, *halp)

  // init DSL state object and parse the input
  dsl := &grammar.DSL2ES{
    Queries:    &utils.QueryStack{},
    Values:     &utils.ValueStack{},
    Verbose:    *verbose,
    IsFilter:   *isFilter,
    Buffer:     *query,
  }
  dsl.Init()
  if err := dsl.Parse(); err != nil {
    log.Fatalf("Parse Error: %s", err)
  }
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
  log.Println(j)
}

func usage() string {
  return fmt.Sprintf("Usage: %s -query \"query string\" [-filter] [-verbose] [-help]", os.Args[0])
  // TODO: detail the DSL grammar etc. here also, or with verbose + help opts together only?
}
